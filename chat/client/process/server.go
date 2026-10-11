package process

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/VKGQQ/goproject_1/chat/client/utils"
	"github.com/VKGQQ/goproject_1/chat/common/message"
)

func ShowMenu() {
	for {
		fmt.Println("----------恭喜您登录成功---------")
		fmt.Println("----------1.显示在线用户列表-----")
		fmt.Println("----------2.发送信息----------")
		fmt.Println("----------3.信息列表----------")
		fmt.Println("----------4.退出系统----------")
		fmt.Println("请选择(1-4):")
		var key int
		var content string
		smsProcess := &SmsProcess{}
		fmt.Scanf("%d\n", &key)
		switch key {
		case 1:
			outputOnlineUser()
		case 2:
			fmt.Println("请输入您想群发的消息")
			fmt.Scanf("%s\n", &content)
			err := smsProcess.SendGroupMes(content)
			if err != nil {
				return
			}
		case 3:
			fmt.Println("信息列表")
		case 4:
			fmt.Println("您选择退出系统")
			os.Exit(0)
		default:
			fmt.Println("您的输入有误请重新输入")
		}
	}
}

func serverProcessMes(Conn net.Conn) {
	tf := &utils.Transfer{
		Conn: Conn,
	}
	for {
		mes, err := tf.ReadPkg()
		if err != nil {
			fmt.Println("tf.ReadPkg err:", err)
			return
		}
		switch mes.Type {
		case message.NotifyUserStatusMesType:
			var notifyUserStatusMes message.NotifyUserStatusMes
			err := json.Unmarshal([]byte(mes.Data), &notifyUserStatusMes)
			if err != nil {
				return
			}
			updateUserStatus(&notifyUserStatusMes)
			outputOnlineUser()
		case message.SmsTransferMesType:
			outputGroupMes(&mes)
		default:
			fmt.Println("服务器返回未知消息类型")
		}
	}
}
