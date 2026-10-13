// Package main 聊天服务端入口：初始化 Redis 与 DAO，监听 0.0.0.0:8889，
// 并为每个客户端连接启动独立的处理协程。
package main

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/VKGQQ/go-chat/chat/server/model"
)

// handleConn 处理单个客户端连接的完整生命周期，直到连接断开。
func handleConn(conn net.Conn) {
	// 单连接兜底：任何意外 panic 只隔离当前连接，不能让整个服务器进程崩溃
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("连接处理异常，已隔离该连接:", r)
		}
		_ = conn.Close()
	}()
	processor := &Processor{
		Conn: conn,
	}
	err := processor.serve()
	if err != nil {
		fmt.Println("客户端与服务器通讯协程出错：", err)
		return
	}
}

// initUserDao 在 Redis 连接就绪后，将各 DAO 实例注入 model 包的全局变量。
// 必须先于任何请求处理调用，否则全局 DAO 为 nil 会直接 panic。
func initUserDao() {
	model.MyUserDao = model.NewUserDao(redisClient)
	model.MyOfflineDao = model.NewOfflineDao(redisClient)
}

// getEnv 读取环境变量，未设置时回退到默认值。
// 默认值保住本地 go run 的开发体验，部署时用环境变量覆盖即可，
// 避免把生产地址/密码这类敏感配置硬编码进代码和 git 历史
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// main 启动流程：连接 Redis（失败即退出）→ 初始化 DAO → 监听端口 → 循环接受连接。
func main() {
	redisAddr := getEnv("REDIS_ADDR", "127.0.0.1:6379")
	redisPwd := getEnv("REDIS_PASSWORD", "")
	if err := initRedis(redisAddr, redisPwd, 0, 20); err != nil {
		fmt.Println("启动失败:", err)
		os.Exit(1)
	}
	// 启动时打印生效配置（密码打码）："连错环境"是部署排查中最高频的问题
	fmt.Printf("Redis 连接配置: addr=%s password=***\n", redisAddr)
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
			// listener 被关闭意味着服务主动关停，继续 Accept 毫无意义，退出主循环
			if errors.Is(err, net.ErrClosed) {
				fmt.Println("服务器停止接受新连接:", err)
				return
			}
			// fd 耗尽等瞬时错误：不能把 nil conn 交给 handleConn，跳过本轮继续服务
			fmt.Println("accept err:", err)
			continue
		}
		go handleConn(conn)
	}
}
