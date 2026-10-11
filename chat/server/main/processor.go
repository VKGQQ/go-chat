package main

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/VKGQQ/goproject_1/chat/common/message"
	"github.com/VKGQQ/goproject_1/chat/server/process"
	"github.com/VKGQQ/goproject_1/chat/server/utils"
)

type Processor struct {
	Conn net.Conn
}

func (this *Processor) serverProcessMes(mes *message.Message) (err error) {
	switch mes.Type {
	case message.LoginMesType:
		up := &process.UserProcess{
			Conn: this.Conn,
		}
		err = up.ServerProcessLogin(mes)
	case message.RegisterMesType:
		up := &process.UserProcess{
			Conn: this.Conn,
		}
		err = up.ServerProcessRegister(mes)
	case message.SmsMesType:
		smsProcess := &process.SmsProcess{}
		smsProcess.SendGroupMes(mes)
	default:
		fmt.Println("消息类型不存在，无法处理......")
	}
	return
}

func (this *Processor) serve() (err error) {
	for {
		tf := &utils.Transfer{
			Conn: this.Conn,
		}
		var mes message.Message
		mes, err = tf.ReadPkg()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("客户端退出，服务端也退出")
				return
			}
			fmt.Println("读包出错，连接异常退出:", err)
			return
		}
		fmt.Printf("mes=%v\n", mes)
		err = this.serverProcessMes(&mes)
		if err != nil {
			fmt.Println("serverProcessMes err:", err)
			return
		}
	}
}
