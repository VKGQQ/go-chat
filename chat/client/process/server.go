package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/VKGQQ/go-chat/chat/client/utils"
	"github.com/VKGQQ/go-chat/chat/common/message"
)

// ShowMenu 登录成功后的聊天主菜单：显示在线用户、群聊、私聊、信息列表、退出。
// 循环阻塞在当前 goroutine，直到用户选择退出进程。
func ShowMenu() {
	smsProcess := &SmsProcess{}
	for {
		fmt.Println("----------恭喜您登录成功---------")
		fmt.Println("----------1.显示在线用户列表-----")
		fmt.Println("----------2.发送群消息----------")
		fmt.Println("----------3.发送私聊----------")
		fmt.Println("----------4.信息列表----------")
		fmt.Println("----------5.退出系统----------")
		fmt.Println("请选择(1-5):")
		key, err := utils.ReadInt()
		if err != nil {
			// 标准输入关闭时退出，避免空转死循环
			if errors.Is(err, io.EOF) {
				fmt.Println()
				os.Exit(0)
			}
			fmt.Println("您的输入有误请重新输入")
			continue
		}
		switch key {
		case 1:
			outputOnlineUser()
		case 2:
			fmt.Println("请输入您想群发的消息")
			content, err := utils.ReadLine()
			if err != nil {
				fmt.Println("读取消息失败，请重新输入")
				continue
			}
			if content == "" {
				fmt.Println("消息不能为空")
				continue
			}
			if err = smsProcess.SendGroupMes(content); err != nil {
				return
			}
		case 3:
			fmt.Println("请输入对方用户ID：")
			targetId, err := utils.ReadInt()
			if err != nil {
				fmt.Println("输入有误，请重新选择")
				continue
			}
			if targetId == CurUser.UserId {
				fmt.Println("不能给自己发私聊")
				continue
			}
			if err = smsProcess.SendPrivateMes(targetId); err != nil {
				fmt.Println("发送私聊失败:", err)
			}
		case 4:
			outputMessageRecords()
		case 5:
			fmt.Println("您选择退出系统")
			os.Exit(0)
		default:
			fmt.Println("您的输入有误请重新输入")
		}
	}
}

// serverProcessMes 接收 goroutine 的主循环：持续读取服务端推送，
// 按类型分发到状态更新、群聊展示或私聊展示；连接断开时退出。
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
		case message.SmsPrivateMesType:
			outputPrivateMes(&mes)
		default:
			fmt.Println("服务器返回未知消息类型")
		}
	}
}
