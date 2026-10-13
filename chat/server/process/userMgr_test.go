package process

import (
	"sync"
	"testing"
)

// newTestUserMgr 为每个测试用例创建独立的 UserMgr 实例，
// 避免全局 userMgr 在用例间互相污染。
func newTestUserMgr() *UserMgr {
	return &UserMgr{onlineUsers: make(map[int]*UserProcess)}
}

func TestAddOnlineUser(t *testing.T) {
	mgr := newTestUserMgr()
	up := &UserProcess{UserId: 100}

	if old := mgr.AddOnlineUser(up); old != nil {
		t.Fatalf("首次加入不应返回旧连接, got %v", old)
	}
	got, err := mgr.GetOnlineUserById(100)
	if err != nil || got != up {
		t.Fatalf("期望找回刚加入的连接, got=%v err=%v", got, err)
	}
}

func TestAddOnlineUser_DuplicateLoginReturnsOld(t *testing.T) {
	mgr := newTestUserMgr()
	old := &UserProcess{UserId: 100}
	newConn := &UserProcess{UserId: 100}

	mgr.AddOnlineUser(old)
	// 重复登录：返回旧连接供调用方互踢，表中替换为新连接
	if kicked := mgr.AddOnlineUser(newConn); kicked != old {
		t.Fatalf("重复登录应返回旧连接, got %v", kicked)
	}
	got, err := mgr.GetOnlineUserById(100)
	if err != nil || got != newConn {
		t.Fatalf("表中应是新连接, got=%v err=%v", got, err)
	}
}

func TestDelOnlineUserIfMatch(t *testing.T) {
	old := &UserProcess{UserId: 100}
	newConn := &UserProcess{UserId: 100}

	tests := []struct {
		name string
		// setup 把场景准备好的连接放进表里
		setup   func(mgr *UserMgr)
		delId   int
		delConn *UserProcess
		wantDel bool
	}{
		{"本人连接正常删除", func(mgr *UserMgr) { mgr.AddOnlineUser(old) }, 100, old, true},
		// 互踢场景：旧连接断开触发清理时，表中已是新连接，旧连接无权删除
		{"被踢旧连接无权删除新连接", func(mgr *UserMgr) { mgr.AddOnlineUser(old); mgr.AddOnlineUser(newConn) }, 100, old, false},
		{"删除不存在的用户", func(mgr *UserMgr) {}, 999, old, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newTestUserMgr()
			tt.setup(mgr)
			if got := mgr.DelOnlineUserIfMatch(tt.delId, tt.delConn); got != tt.wantDel {
				t.Errorf("want %v, got %v", tt.wantDel, got)
			}
		})
	}
}

func TestGetAllOnlineUsers_SnapshotIsolation(t *testing.T) {
	mgr := newTestUserMgr()
	mgr.AddOnlineUser(&UserProcess{UserId: 100})

	snapshot := mgr.GetAllOnlineUsers()
	delete(snapshot, 100) // 篡改快照不应影响内部表

	if got := len(mgr.GetAllOnlineUsers()); got != 1 {
		t.Fatalf("快照被篡改后内部表应保持 1 个用户, got %d", got)
	}
}

func TestGetOnlineUserById_NotFound(t *testing.T) {
	mgr := newTestUserMgr()
	if _, err := mgr.GetOnlineUserById(404); err == nil {
		t.Fatal("查询不存在的用户应返回错误")
	}
}

// TestUserMgr_ConcurrentAccess 并发增删读，依赖 -race 检测器验证锁保护是否完备。
func TestUserMgr_ConcurrentAccess(t *testing.T) {
	mgr := newTestUserMgr()
	const n = 50
	ups := make([]*UserProcess, n)
	for i := range ups {
		ups[i] = &UserProcess{UserId: i + 1}
	}

	var wg sync.WaitGroup
	for _, up := range ups {
		wg.Add(2)
		go func(u *UserProcess) { defer wg.Done(); mgr.AddOnlineUser(u) }(up)
		go func() {
			defer wg.Done()
			_ = mgr.GetAllOnlineUsers()
			_ = mgr.OnlineUserIds()
			_, _ = mgr.GetOnlineUserById(1)
		}()
	}
	wg.Wait()

	if got := len(mgr.GetAllOnlineUsers()); got != n {
		t.Fatalf("并发加入后应有 %d 个用户, got %d", n, got)
	}

	for _, up := range ups {
		wg.Add(1)
		go func(u *UserProcess) { defer wg.Done(); mgr.DelOnlineUserIfMatch(u.UserId, u) }(up)
	}
	wg.Wait()
	if got := len(mgr.GetAllOnlineUsers()); got != 0 {
		t.Fatalf("全部删除后应为空表, got %d", got)
	}
}
