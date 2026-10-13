package process

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/VKGQQ/goproject_1/chat/common/message"
	"github.com/VKGQQ/goproject_1/chat/server/model"
)

type SmsProcess struct {
}

// SendGroupMes 群发消息。senderId 是服务端登录会话确认的权威发送者身份：
// 强制覆盖客户端上报的 UserId，防止伪造他人发言。
func (this *SmsProcess) SendGroupMes(mes *message.Message, senderId int) {

	//遍历服务器端的onlineUsers map[int]*UserProcess,
	//将消息转发取出.
	//取出mes的内容 SmsMes
	var smsMes message.SmsMes
	err := json.Unmarshal([]byte(mes.Data), &smsMes)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err)
		return
	}
	// 以服务端会话身份为准重新序列化，转发的消息中不再含客户端伪造字段
	smsMes.UserId = senderId
	smsData, err := json.Marshal(smsMes)
	if err != nil {
		fmt.Println("json.Marshal err=", err)
		return
	}
	mes.Data = string(smsData)
	mes.Type = message.SmsTransferMesType
	data, err := json.Marshal(mes)
	if err != nil {
		fmt.Println("json.Marshal err=", err)
		return
	}

	// 基于快照在锁外遍历转发，避免持读锁做网络 IO
	for id, up := range userMgr.GetAllOnlineUsers() {
		//这里，还需要过滤到自己,即不要再发给自己（senderId 不可伪造）
		if id == senderId {
			continue
		}
		// 走收信人自己的写锁，保证帧完整
		if err := up.Send(data); err != nil {
			fmt.Println("转发消息失败 err=", err)
		}
	}
}

// SendPrivateMes 私聊转发。sender 是发送者的会话实例：
// 既提供权威发送者身份（和群聊同理防伪造），又能在目标不在线时回通知。
func (this *SmsProcess) SendPrivateMes(mes *message.Message, sender *UserProcess) {
	var smsPrivateMes message.SmsPrivateMes
	err := json.Unmarshal([]byte(mes.Data), &smsPrivateMes)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err)
		return
	}
	// 身份覆盖：客户端填的 UserId 不可信，以服务端登录会话为准
	smsPrivateMes.UserId = sender.UserId

	// 精准路由：查在线表找目标
	targetUp, err := userMgr.GetOnlineUserById(smsPrivateMes.TargetId)
	if err != nil {
		// 目标不在线：存离线消息队列，等对方上线后按发送顺序投递
		offlineMsg := &model.OfflineMessage{
			SenderId:  sender.UserId,
			TargetId:  smsPrivateMes.TargetId,
			Content:   smsPrivateMes.Content,
			Timestamp: time.Now(),
		}
		if storeErr := model.MyOfflineDao.StoreOfflineMessage(offlineMsg); storeErr != nil {
			fmt.Println("存离线消息失败 err=", storeErr)
		}
		// 回系统提示给发送者
		notify := message.SmsPrivateMes{
			Content: fmt.Sprintf("用户 %d 不在线，消息已保存，将在其上线后送达", smsPrivateMes.TargetId),
		}
		notifyData, _ := json.Marshal(notify)
		var notifyMes message.Message
		notifyMes.Type = message.SmsPrivateMesType
		notifyMes.Data = string(notifyData)
		data, _ := json.Marshal(notifyMes)
		if err := sender.Send(data); err != nil {
			fmt.Println("发送私聊通知失败 err=", err)
		}
		return
	}

	// 重新序列化（用覆盖后的权威身份），转发给目标
	smsData, err := json.Marshal(smsPrivateMes)
	if err != nil {
		fmt.Println("json.Marshal err=", err)
		return
	}
	mes.Data = string(smsData)
	data, err := json.Marshal(mes)
	if err != nil {
		fmt.Println("json.Marshal err=", err)
		return
	}
	if err := targetUp.Send(data); err != nil {
		fmt.Println("私聊转发失败 err=", err)
	}
}
