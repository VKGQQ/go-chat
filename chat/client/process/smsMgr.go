package process

import (
	"fmt"

	"encoding/json"

	"github.com/VKGQQ/goproject_1/chat/common/message"
)

func outputGroupMes(mes *message.Message) {
	//反序列化mes.Data
	var smsMes message.SmsMes
	err := json.Unmarshal([]byte(mes.Data), &smsMes)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err.Error())
		return
	}

	//显示信息
	info := fmt.Sprintf("接收到用户id:\t%d 发来的信息:\t%s",
		smsMes.UserId, smsMes.Content)
	fmt.Println(info)
	fmt.Println()
}

func outputPrivateMes(mes *message.Message) {
	var smsPrivateMes message.SmsPrivateMes
	err := json.Unmarshal([]byte(mes.Data), &smsPrivateMes)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err.Error())
		return
	}
	// UserId == 0 是服务端发来的系统提示（如"用户不在线"）
	if smsPrivateMes.UserId == 0 {
		fmt.Printf("[系统信息] %s\n\n", smsPrivateMes.Content)
		return
	}
	info := fmt.Sprintf("[私聊信息] 用户%d说：\t%s",
		smsPrivateMes.UserId, smsPrivateMes.Content)
	fmt.Println(info)
	fmt.Println()
}
