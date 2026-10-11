package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/VKGQQ/goproject_1/chat/client/process"
	"github.com/VKGQQ/goproject_1/chat/client/utils"
)

var userId int
var userPwd string
var userName string

func main() {
	for {
		fmt.Println("----------欢迎登录多人聊天系统----------")
		fmt.Println("\t 1 登录聊天室")
		fmt.Println("\t 2 注册用户")
		fmt.Println("\t 3 退出系统")
		fmt.Printf("请选择(1-3):")
		key, err := utils.ReadInt()
		if err != nil {
			// 标准输入关闭（Ctrl+Z 再回车）时直接退出，避免空转死循环
			if errors.Is(err, io.EOF) {
				fmt.Println()
				os.Exit(0)
			}
			fmt.Println("您的输入有误请重新输入")
			continue
		}
		switch key {
		case 1:
			fmt.Println("请输入用户的id")
			userId, err = utils.ReadInt()
			if err != nil {
				fmt.Println("用户id输入有误，请重新输入")
				continue
			}
			fmt.Println("请输入用户的密码")
			userPwd, err = utils.ReadLine()
			if err != nil {
				fmt.Println("读取密码失败，请重新输入")
				continue
			}
			up := &process.UserProcess{}
			err = up.Login(userId, userPwd)
			if err != nil {
				return
			}
		case 2:
			fmt.Println("请输入用户ID：")
			userId, err = utils.ReadInt()
			if err != nil {
				fmt.Println("用户ID输入有误，请重新输入")
				continue
			}
			fmt.Println("请输入用户密码：")
			userPwd, err = utils.ReadLine()
			if err != nil {
				fmt.Println("读取密码失败，请重新输入")
				continue
			}
			fmt.Println("请输入用户昵称：")
			userName, err = utils.ReadLine()
			if err != nil {
				fmt.Println("读取昵称失败，请重新输入")
				continue
			}
			up := &process.UserProcess{}
			err = up.Register(userId, userPwd, userName)
			if err != nil {
				return
			}
		case 3:
			os.Exit(0)
		default:
			fmt.Println("您的输入有误请重新输入")
		}
	}
}
