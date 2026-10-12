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
	// up 是本连接唯一的会话：登录成功后 UserId 有效，
	// 连接结束时据此清理在线表
	up *process.UserProcess
}

func (this *Processor) serverProcessMes(mes *message.Message) (err error) {
	switch mes.Type {
	case message.LoginMesType:
		err = this.up.ServerProcessLogin(mes)
	case message.RegisterMesType:
		err = this.up.ServerProcessRegister(mes)
	case message.SmsMesType:
		// 未登录连接（UserId == 0）无权群发
		if this.up.UserId == 0 {
			fmt.Println("未登录连接尝试群发，已拒绝")
			return
		}
		smsProcess := &process.SmsProcess{}
		// 发送者身份以服务端登录会话为准，客户端上报的一律不信
		smsProcess.SendGroupMes(mes, this.up.UserId)
	default:
		fmt.Println("消息类型不存在，无法处理......")
	}
	return
}

func (this *Processor) serve() (err error) {
	// 每条连接只创建一个 UserProcess，登录后身份保存在其中
	this.up = &process.UserProcess{
		Conn: this.Conn,
	}
	// 连接结束的唯一清理出口：正常 EOF、读错误、处理错误、panic 都会走到这里。
	// 必须在 handleConn 的 recover 内层注册，panic 时也能先完成下线清理。
	defer func() {
		process.UserOffline(this.up)
	}()
	for {
		tf := &utils.Transfer{
			Conn: this.Conn,
		}
		var mes message.Message
		mes, err = tf.ReadPkg()
		if err != nil {
			// EOF：对端在一帧开始前正常关闭；ErrUnexpectedEOF：帧读到一半对端断开。
			// 两者都视为连接结束，serve 返回后由 defer 统一清理。
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
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
