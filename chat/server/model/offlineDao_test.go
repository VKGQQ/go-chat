package model

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newTestOfflineDao 每个用例使用独立的 miniredis 实例。
func newTestOfflineDao(t *testing.T) *OfflineDao {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewOfflineDao(client)
}

func offlineMsg(senderId, targetId int, content string) *OfflineMessage {
	return &OfflineMessage{
		SenderId:  senderId,
		TargetId:  targetId,
		Content:   content,
		Timestamp: time.Now(),
	}
}

func TestOfflineDao_StoreAndPopFifo(t *testing.T) {
	dao := newTestOfflineDao(t)

	// 按序存 3 条，必须按相同顺序弹出（RPUSH + LPOP 构成 FIFO）
	for _, content := range []string{"first", "second", "third"} {
		if err := dao.StoreOfflineMessage(offlineMsg(1, 100, content)); err != nil {
			t.Fatalf("存离线消息失败: %v", err)
		}
	}
	for _, want := range []string{"first", "second", "third"} {
		msg, err := dao.PopOfflineMessage(100)
		if err != nil || msg == nil {
			t.Fatalf("弹出失败: msg=%v err=%v", msg, err)
		}
		if msg.Content != want {
			t.Errorf("FIFO 顺序破坏: want %q, got %q", want, msg.Content)
		}
	}
	// 队列已空：返回 (nil, nil) 而非错误
	msg, err := dao.PopOfflineMessage(100)
	if msg != nil || err != nil {
		t.Errorf("空队列应返回 (nil, nil), got msg=%v err=%v", msg, err)
	}
}

func TestOfflineDao_PopIsolatedByTargetUser(t *testing.T) {
	dao := newTestOfflineDao(t)

	_ = dao.StoreOfflineMessage(offlineMsg(1, 100, "给100"))
	msg, err := dao.PopOfflineMessage(200)
	if msg != nil || err != nil {
		t.Errorf("其他用户的队列应为空, got msg=%v err=%v", msg, err)
	}
}

// TestOfflineDao_RequeueKeepsOrder 投递失败推回队首后，原有 FIFO 顺序必须保持。
func TestOfflineDao_RequeueKeepsOrder(t *testing.T) {
	dao := newTestOfflineDao(t)

	_ = dao.StoreOfflineMessage(offlineMsg(1, 100, "A"))
	_ = dao.StoreOfflineMessage(offlineMsg(1, 100, "B"))

	// 弹出 A 后投递失败，推回队首；再弹必须还是 A
	msg, _ := dao.PopOfflineMessage(100)
	if err := dao.RequeueOfflineMessage(msg); err != nil {
		t.Fatalf("推回队首失败: %v", err)
	}
	for _, want := range []string{"A", "B"} {
		msg, err := dao.PopOfflineMessage(100)
		if err != nil || msg == nil {
			t.Fatalf("弹出失败: msg=%v err=%v", msg, err)
		}
		if msg.Content != want {
			t.Errorf("推回后顺序破坏: want %q, got %q", want, msg.Content)
		}
	}
}
