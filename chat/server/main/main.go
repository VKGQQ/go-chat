package main

import (
	"fmt"
	"net"
	"os"

	"github.com/VKGQQ/goproject_1/chat/server/model"
)

func handleConn(conn net.Conn) {
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)
	processor := &Processor{
		Conn: conn,
	}
	err := processor.serve()
	if err != nil {
		fmt.Println("客户端与服务器通讯协程出错：", err)
		return
	}
}

func initUserDao() {
	model.MyUserDao = model.NewUserDao(redisClient)
}

func main() {
	if err := initRedis("127.0.0.1:6379", "", 0, 20); err != nil {
		fmt.Println("启动失败:", err)
		os.Exit(1)
	}
	initUserDao()
	fmt.Println("服务器在8889端口监听")
	listen, err := net.Listen("tcp", "0.0.0.0:8889")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func(listen net.Listener) {
		_ = listen.Close()
	}(listen)
	for {
		fmt.Println("等待客户端连接服务器......")
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println(err)
		}
		go handleConn(conn)
	}
}
