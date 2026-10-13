package process

import (
	"fmt"
	"sync"

	"github.com/VKGQQ/goproject_1/chat/client/model"
	"github.com/VKGQQ/goproject_1/chat/common/message"
)

// onlineUsers 会被两个 goroutine 并发访问：
// 接收 goroutine（updateUserStatus/addOnlineUser 写）与菜单 goroutine
// （outputOnlineUser 遍历读），必须用锁保护，否则并发读写直接崩进程。
var (
	onlineUsers = make(map[int]*message.User, 20)
	onlineMu    sync.RWMutex
)

// CurUser 当前登录用户的会话，登录成功后由 UserProcess.Login 填充。
var CurUser model.CurUser

// outputOnlineUser 打印当前在线用户列表（菜单“显示在线用户列表”）。
func outputOnlineUser() {
	onlineMu.RLock()
	defer onlineMu.RUnlock()
	fmt.Println("当前在线用户列表：")
	for id := range onlineUsers {
		fmt.Println("用户ID：\t", id)
	}
	fmt.Println()
}

// addOnlineUser 登录成功后批量写入在线列表（写锁）
func addOnlineUser(userId int) {
	onlineMu.Lock()
	defer onlineMu.Unlock()
	onlineUsers[userId] = &message.User{
		UserId:     userId,
		UserStatus: message.UserOnline,
	}
}

// updateUserStatus 根据服务端广播更新本地在线列表（写锁）：
// 下线直接删除表项，上线则新增或更新状态。
func updateUserStatus(notifyUserStatusMes *message.NotifyUserStatusMes) {
	onlineMu.Lock()
	defer onlineMu.Unlock()
	// 下线用户直接从本地列表删除，否则列表堆积僵尸用户，
	// 且 outputOnlineUser 不区分状态会把已下线用户也显示出来
	if notifyUserStatusMes.Status == message.UserOffline {
		delete(onlineUsers, notifyUserStatusMes.UserId)
		return
	}
	user, ok := onlineUsers[notifyUserStatusMes.UserId]
	if !ok {
		user = &message.User{
			UserId: notifyUserStatusMes.UserId,
		}
	}
	user.UserStatus = notifyUserStatusMes.Status
	onlineUsers[notifyUserStatusMes.UserId] = user
}
