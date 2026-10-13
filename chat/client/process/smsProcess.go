package process

import (
	"fmt"

	"encoding/json"

	"github.com/VKGQQ/goproject_1/chat/client/utils"
	"github.com/VKGQQ/goproject_1/chat/common/message"
)

type SmsProcess struct {
}

//发送群聊的消息

func (this *SmsProcess) SendGroupMes(content string) (err error) {

	//1 创建一个Mes
	var mes message.Message
	mes.Type = message.SmsMesType

	//2 创建一个SmsMes 实例
	var smsMes message.SmsMes
	smsMes.Content = content               //内容.
	smsMes.UserId = CurUser.UserId         //
	smsMes.UserStatus = CurUser.UserStatus //

	//3.序列化 smsMes
	data, err := json.Marshal(smsMes)
	if err != nil {
		fmt.Println("SendGroupMes json.Marshal fail =", err.Error())
		return
	}

	mes.Data = string(data)

	//4. 对mes再次序列化
	data, err = json.Marshal(mes)
	if err != nil {
		fmt.Println("SendGroupMes json.Marshal fail =", err.Error())
		return
	}

	//5. 将mes发送给服务器。。
	tf := &utils.Transfer{
		Conn: CurUser.Conn,
	}
	//6.发送
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
