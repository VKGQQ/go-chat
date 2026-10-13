package process

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/VKGQQ/goproject_1/chat/common/message"
	"github.com/VKGQQ/goproject_1/chat/server/model"
)

// connWriteTimeout 单次发包（含帧头和帧体）的写超时时间。
// 必须设置：否则对端网络卡住时 Conn.Write 会无限期阻塞，
// 持有的 writeMu 也会一直不释放，导致其他 goroutine 无法给该用户发包。
const connWriteTimeout = 5 * time.Second

type UserProcess struct {
	Conn   net.Conn
	UserId int
	// writeMu 是“每连接一把”的写锁：同一连接的帧头+帧体必须在
	// 持锁期间连续写出，保证多个 goroutine（本人响应、群发转发、上线通知）
	// 并发发包时不会出现字节交错导致的帧失步。
	// 不同连接各有一把锁，互不影响、可完全并行。
	writeMu sync.Mutex
}

// Send 是向该用户连接写入一个完整协议帧的唯一入口。
// 帧格式：4 字节大端长度 + JSON 包体。
func (this *UserProcess) Send(data []byte) error {
	this.writeMu.Lock()
	defer this.writeMu.Unlock()
	// 每次发包前设置绝对截止时间，避免慢客户端把锁长期占住
	if err := this.Conn.SetWriteDeadline(time.Now().Add(connWriteTimeout)); err != nil {
		return err
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(data)))
	if _, err := this.Conn.Write(header[:]); err != nil {
		return err
	}
	if _, err := this.Conn.Write(data); err != nil {
		return err
	}
	return nil
}

// broadcastUserStatus 向除 userId 外的所有在线用户广播其上/下线状态。
// 基于快照在锁外遍历发送，避免持读锁做网络 IO。
func broadcastUserStatus(userId int, status int) {
	for id, up := range userMgr.GetAllOnlineUsers() {
		if id == userId {
			continue
		}
		up.NotifyMeStatus(userId, status)
	}
}

// UserOffline 是连接结束时的统一清理入口：
// 先移出在线表（立即阻止后续群发再写该死连接），再向其他在线用户广播下线。
// 未登录（UserId == 0）的连接断开时无需清理。
// 互踢场景：被踢的旧连接断开时表中已是新连接，IfMatch 删除失败，
// 既不能误删新表项，也不能误广播“下线”（该用户其实还在线）。
func UserOffline(up *UserProcess) {
	if up == nil || up.UserId == 0 {
		return
	}
	if !userMgr.DelOnlineUserIfMatch(up.UserId, up) {
		return
	}
	fmt.Printf("用户 %d 已下线\n", up.UserId)
	broadcastUserStatus(up.UserId, message.UserOffline)
}

// NotifyMeStatus 通知“当前连接对应的用户”：userId 的状态变为 status（上线/下线）
func (this *UserProcess) NotifyMeStatus(userId int, status int) {
	var mes message.Message
	mes.Type = message.NotifyUserStatusMesType
	var notifyUserStatusMes message.NotifyUserStatusMes
	notifyUserStatusMes.UserId = userId
	notifyUserStatusMes.Status = status
	data, err := json.Marshal(notifyUserStatusMes)
	if err != nil {
		fmt.Println("json.Marshal err:", err)
		return
	}
	mes.Data = string(data)
	data, err = json.Marshal(mes)
	if err != nil {
		fmt.Println("json.Marshal err:", err)
		return
	}
	if err = this.Send(data); err != nil {
		fmt.Println("NotifyMeStatus err:", err)
	}
}

func (this *UserProcess) ServerProcessLogin(mes *message.Message) (err error) {
	var loginMes message.LoginMes
	err = json.Unmarshal([]byte(mes.Data), &loginMes)
	if err != nil {
		fmt.Println("json.Unmarshal err:", err)
		return
	}
	var resMes message.Message
	resMes.Type = message.LoginResMesType
	var loginResMes message.LoginResMes
	user, err := model.MyUserDao.Login(loginMes.UserId, loginMes.UserPwd)
	if err != nil {
		if errors.Is(err, model.ErrorUserNotExists) {
			loginResMes.Code = 500
			loginResMes.Error = err.Error()
		} else if errors.Is(err, model.ErrorUserPwd) {
			loginResMes.Code = 403
			loginResMes.Error = err.Error()
		} else {
			loginResMes.Code = 505
			loginResMes.Error = "服务器内部错误"
		}
	} else {
		loginResMes.Code = 200
		this.UserId = user.UserId
		// 重复登录互踢：同账号已有旧连接时，新连接入表后关闭旧连接。
		// 旧连接的读循环随连接关闭而退出，走各自的下线清理，
		// 因表中已是新连接，旧连接不会误删表项。
		if old := userMgr.AddOnlineUser(this); old != nil {
			fmt.Printf("用户 %d 在别处登录，踢掉旧连接\n", this.UserId)
			_ = old.Conn.Close()
		}
		broadcastUserStatus(this.UserId, message.UserOnline)
		loginResMes.UsersId = userMgr.OnlineUserIds()
		fmt.Println(user, "登录成功")
	}
	data, err := json.Marshal(loginResMes)
	if err != nil {
		fmt.Println("json.Marshal err:", err)
		return
	}
	resMes.Data = string(data)
	data, err = json.Marshal(resMes)
	if err != nil {
		fmt.Println("json.Marshal err:", err)
		return
	}
	err = this.Send(data)
	// 登录响应发出后再异步投递离线消息：不阻塞登录，
	// 用户先看到自己上线，离线消息随后陆续到达。
	if err == nil && loginResMes.Code == 200 {
		go this.deliverOfflineMessages(this.UserId)
	}
	return
}

// deliverOfflineMessages 登录成功后异步投递离线消息。
// LPOP 逐条“取出即删”，投递失败推回队首并停止，等下次上线再投；
// 消息包装成 SmsPrivateMesType，客户端现有接收逻辑直接处理，无需改动。
func (this *UserProcess) deliverOfflineMessages(userId int) {
	for {
		msg, err := model.MyOfflineDao.PopOfflineMessage(userId)
		if err != nil {
			fmt.Println("读取离线消息失败 err=", err)
			return
		}
		if msg == nil {
			return // 没有更多离线消息
		}
		smsPrivateMes := message.SmsPrivateMes{
			TargetId: userId,
			Content:  msg.Content,
		}
		smsPrivateMes.UserId = msg.SenderId
		smsData, err := json.Marshal(smsPrivateMes)
		if err != nil {
			fmt.Println("json.Marshal err=", err)
			return
		}
		var mes message.Message
		mes.Type = message.SmsPrivateMesType
		mes.Data = string(smsData)
		data, err := json.Marshal(mes)
		if err != nil {
			fmt.Println("json.Marshal err=", err)
			return
		}
		if err = this.Send(data); err != nil {
			// 连接可能已断：推回队首保序，停止本轮投递
			_ = model.MyOfflineDao.RequeueOfflineMessage(msg)
			return
		}
	}
}

func (this *UserProcess) ServerProcessRegister(mes *message.Message) (err error) {
	var registerMes message.RegisterMes
	err = json.Unmarshal([]byte(mes.Data), &registerMes)
	if err != nil {
		fmt.Println("json.Unmarshal err:", err)
		return
	}
	var resMes message.Message
	resMes.Type = message.RegisterResMesType
	var registerResMes message.RegisterResMes
	// 系统边界校验：客户端输入不可信，任何人都能用裸 TCP 发包，
	// 非法数据（0 值 id、空密码、空昵称）必须在入库前拦截。
	// 0 同时是“未登录连接”的哨兵值，注册 0 会污染在线状态判断。
	if registerMes.User.UserId <= 0 || registerMes.User.UserPwd == "" || registerMes.User.UserName == "" {
		registerResMes.Code = 507
		registerResMes.Error = "注册信息不合法：id必须为正整数，密码和昵称不能为空"
	} else if err = model.MyUserDao.Register(&registerMes.User); err != nil {
		if errors.Is(err, model.ErrorUserExists) {
			registerResMes.Code = 505
			registerResMes.Error = err.Error()
		} else {
			registerResMes.Code = 506
			registerResMes.Error = "注册时发生未知错误"
		}
	} else {
		registerResMes.Code = 200
	}
	data, err := json.Marshal(registerResMes)
	if err != nil {
		fmt.Println("json.Marshal err:", err)
		return
	}
	resMes.Data = string(data)
	data, err = json.Marshal(resMes)
	if err != nil {
		fmt.Println("json.Marshal err:", err)
		return
	}
	err = this.Send(data)
	return
}
