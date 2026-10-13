package process

import (
	"fmt"
	"maps"
	"sync"
)

// userMgr 全局唯一的在线用户管理器，进程内单例，由 init 初始化。
var (
	userMgr *UserMgr
)

// UserMgr 在线用户管理器：维护 userId -> 连接会话 的映射。
type UserMgr struct {
	// onlineUsers 会被多个连接的 goroutine 并发读写，必须用读写锁保护。
	// 读多写少（转发/查询频繁，上下线低频），因此选用 RWMutex。
	mu          sync.RWMutex
	onlineUsers map[int]*UserProcess
}

// init 初始化全局在线用户表，进程启动时自动执行。
func init() {
	userMgr = &UserMgr{
		onlineUsers: make(map[int]*UserProcess, 1024),
	}
}

// AddOnlineUser 将新连接写入在线表（写锁，互斥）。
// 若同一 UserId 已有旧连接（重复登录），在锁内完成“查旧→放新”后返回旧连接，
// 由调用方在锁外关闭（互踢），避免持锁做 IO。
func (this *UserMgr) AddOnlineUser(up *UserProcess) (old *UserProcess) {
	this.mu.Lock()
	defer this.mu.Unlock()
	old = this.onlineUsers[up.UserId]
	this.onlineUsers[up.UserId] = up
	return
}

// DelOnlineUserIfMatch 仅当在线表中 userId 对应的连接就是 up 本人时才删除（写锁）。
// 互踢场景下，被踢的旧连接断开时也会触发下线清理，但此时表中已是新连接，
// 旧连接无权删除。返回是否真正删除，供调用方决定是否需要广播下线。
func (this *UserMgr) DelOnlineUserIfMatch(userId int, up *UserProcess) (removed bool) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if cur, ok := this.onlineUsers[userId]; ok && cur == up {
		delete(this.onlineUsers, userId)
		removed = true
	}
	return
}

// OnlineUserIds 返回当前所有在线用户 id 的快照切片（读锁）
// 锁内完成拷贝，调用方可在锁外安全遍历
func (this *UserMgr) OnlineUserIds() []int {
	this.mu.RLock()
	defer this.mu.RUnlock()
	ids := make([]int, 0, len(this.onlineUsers))
	for id := range this.onlineUsers {
		ids = append(ids, id)
	}
	return ids
}

// GetAllOnlineUsers 返回在线用户表的快照副本（读锁）
// 必须返回拷贝而不能直接返回原 map：否则调用方在锁外遍历时，
// 其他 goroutine 可能正在并发写 map，依然会 panic。
// 网络发送等耗时操作基于快照在锁外进行，避免持锁做 IO 阻塞上下线。
func (this *UserMgr) GetAllOnlineUsers() map[int]*UserProcess {
	this.mu.RLock()
	defer this.mu.RUnlock()
	snapshot := make(map[int]*UserProcess, len(this.onlineUsers))
	maps.Copy(snapshot, this.onlineUsers)
	return snapshot
}

// GetOnlineUserById 根据id返回对应的在线用户（读锁）
func (this *UserMgr) GetOnlineUserById(userId int) (up *UserProcess, err error) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	up, ok := this.onlineUsers[userId]
	if !ok {
		err = fmt.Errorf("用户%d 不存在", userId)
		return
	}
	return
}
