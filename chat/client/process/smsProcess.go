package process

import (
	"fmt"

	"encoding/json"

	"github.com/VKGQQ/goproject_1/chat/client/utils"
	"github.com/VKGQQ/goproject_1/chat/common/message"
)

// SmsProcess 消息发送方：负责群聊与私聊消息的封装和发送。
type SmsProcess struct {
}

// SendGroupMes 发送群聊消息。
// 发送者 UserId 会被服务端按登录会话强制覆盖，这里填充仅为结构完整。
func (this *SmsProcess) SendGroupMes(content string) (err error) {
	var mes message.Message
	mes.Type = message.SmsMesType

	var smsMes message.SmsMes
	smsMes.Content = content
	smsMes.UserId = CurUser.UserId
	smsMes.UserStatus = CurUser.UserStatus

	data, err := json.Marshal(smsMes)
	if err != nil {
		fmt.Println("SendGroupMes json.Marshal fail =", err.Error())
		return
	}
	mes.Data = string(data)
	data, err = json.Marshal(mes)
	if err != nil {
		fmt.Println("SendGroupMes json.Marshal fail =", err.Error())
		return
	}

	tf := &utils.Transfer{
		Conn: CurUser.Conn,
	}
	err = tf.WritePkg(data)
	if err != nil {
		fmt.Println("SendGroupMes err=", err.Error())
		return
	}
	return
}

// SendPrivateMes 发送私聊消息。targetId 是接收者用户ID。
// 发送者 UserId 会被服务端覆盖（和群聊同理），客户端填了也无妨。
func (this *SmsProcess) SendPrivateMes(targetId int) (err error) {
	var mes message.Message
	mes.Type = message.SmsPrivateMesType

	var smsPrivateMes message.SmsPrivateMes
	smsPrivateMes.UserId = CurUser.UserId // 会被服务端覆盖
	smsPrivateMes.TargetId = targetId
	fmt.Println("请输入消息内容：")
	smsPrivateMes.Content, err = utils.ReadLine()
	if err != nil {
		fmt.Println("读取消息失败，请重新输入")
		return
	}
	if smsPrivateMes.Content == "" {
		fmt.Println("消息不能为空")
		return
	}

	data, err := json.Marshal(smsPrivateMes)
	if err != nil {
		fmt.Println("json.Marshal err=", err)
		return
	}
	mes.Data = string(data)
	data, err = json.Marshal(mes)
	if err != nil {
		fmt.Println("json.Marshal err=", err)
		return
	}

	tf := &utils.Transfer{
		Conn: CurUser.Conn,
	}
	err = tf.WritePkg(data)
	if err != nil {
		fmt.Println("SendPrivateMes err=", err)
		return
	}
	return
}
