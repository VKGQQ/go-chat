// Package model 定义客户端本地维护的数据模型。
package model

import (
	"net"

	"github.com/VKGQQ/goproject_1/chat/common/message"
)

// CurUser 当前登录用户的会话：底层连接 + 账号信息。
type CurUser struct {
	Conn net.Conn
	message.User
}
