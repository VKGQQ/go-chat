// Package process 实现客户端核心业务逻辑：登录/注册、消息收发、
// 在线用户列表维护和本地聊天记录持久化。
package process

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/VKGQQ/go-chat/chat/client/utils"
	"github.com/VKGQQ/go-chat/chat/common/message"
)

// UserProcess 用户账号相关请求（登录、注册）的发起方。
type UserProcess struct{}

// serverAddr 服务器地址。0.0.0.0 只能用于监听，客户端拨号必须填具体地址；
// 默认本地地址，部署时通过 CHAT_SERVER_ADDR 环境变量覆盖，无需改代码重新编译
var serverAddr = getEnv("CHAT_SERVER_ADDR", "127.0.0.1:8889")

// getEnv 读取环境变量，未设置时回退到默认值
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Login 发起登录请求。成功后初始化 CurUser 会话、加载本地聊天记录、
// 启动服务端消息接收 goroutine，并进入聊天主菜单（阻塞直至退出）。
func (this *UserProcess) Login(userId int, userPwd string) (err error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return err
	}
	// 连接所有权：登录成功前归本函数管，任何失败路径都要关；
	// 登录成功后所有权移交给 CurUser 与接收 goroutine，本函数不再关闭。
	success := false
	defer func() {
		if !success {
			_ = conn.Close()
		}
	}()
	var mes message.Message
	mes.Type = message.LoginMesType
	var loginMes message.LoginMes
	loginMes.UserId = userId
	loginMes.UserPwd = userPwd

	data, err := json.Marshal(loginMes)
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
	tf := &utils.Transfer{
		Conn: conn,
	}
	err = tf.WritePkg(data)
	if err != nil {
		fmt.Println("登录发送信息错误:", err)
		return
	}
	mes, err = tf.ReadPkg()
	if err != nil {
		fmt.Println("readPkg err:", err)
		return
	}
	var loginResMes message.LoginResMes
	err = json.Unmarshal([]byte(mes.Data), &loginResMes)
	if err != nil {
		fmt.Println("解析登录响应失败:", err)
		return
	}
	if loginResMes.Code == 200 {
		success = true // 交接完成：连接转由 CurUser 与接收 goroutine 持有
		CurUser.Conn = conn
		CurUser.UserId = userId
		CurUser.UserStatus = message.UserOnline
		fmt.Println("当前在线用户ID列表：")
		v := 0
		for _, id := range loginResMes.UsersId {
			if id == userId {
				continue
			}
			v++
			fmt.Printf("%v.%v\n", v, id)
			addOnlineUser(id)
		}
		fmt.Printf("当前还有%v人在线\n\n", v)
		// 先加载本地历史再启动接收 goroutine：
		// 保证旧记录在前、新消息在后，避免新消息插到历史前面的乱序
		loadHistory(userId)
		go serverProcessMes(conn)
		ShowMenu()
	} else {
		fmt.Println(loginResMes.Error)
	}
	return
}

// Register 发起注册请求，打印服务端返回的结果后返回主菜单。
func (this *UserProcess) Register(userId int, userPwd string, userName string) (err error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return err
	}
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)
	var mes message.Message
	mes.Type = message.RegisterMesType
	var registerMes message.RegisterMes
	registerMes.User.UserId = userId
	registerMes.User.UserPwd = userPwd
	registerMes.User.UserName = userName
	data, err := json.Marshal(registerMes)
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
	tf := &utils.Transfer{
		Conn: conn,
	}
	err = tf.WritePkg(data)
	if err != nil {
		fmt.Println("注册发送信息错误:", err)
		return
	}
	mes, err = tf.ReadPkg()
	if err != nil {
		fmt.Println("readPkg err:", err)
		return
	}
	var registerResMes message.RegisterResMes
	err = json.Unmarshal([]byte(mes.Data), &registerResMes)
	if err != nil {
		fmt.Println("解析注册响应失败:", err)
		return
	}
	if registerResMes.Code == 200 {
		fmt.Println("注册成功，请重新登录。")
	} else {
		fmt.Println(registerResMes.Error)
	}
	return
}
