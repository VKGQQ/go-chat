package process

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/VKGQQ/goproject_1/chat/client/utils"
	"github.com/VKGQQ/goproject_1/chat/common/message"
)

type UserProcess struct{}

func (this *UserProcess) Login(userId int, userPwd string) (err error) {
	conn, err := net.Dial("tcp", "0.0.0.0:8889")
	if err != nil {
		return err
	}
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)
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
			user := message.User{
				UserId:     id,
				UserStatus: message.UserOnline,
			}
			onlineUsers[id] = &user
		}
		fmt.Printf("当前还有%v人在线\n\n", v)
		go serverProcessMes(conn)
		ShowMenu()
	} else {
		fmt.Println(loginResMes.Error)
	}
	return
}

func (this *UserProcess) Register(userId int, userPwd string, userName string) (err error) {
	conn, err := net.Dial("tcp", "0.0.0.0:8889")
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
