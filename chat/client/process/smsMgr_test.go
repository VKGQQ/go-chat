package process

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

// resetRecords 清空全局消息记录，并在用例结束后恢复，避免用例间污染。
func resetRecords(t *testing.T) {
	t.Helper()
	msgMu.Lock()
	messageRecords = nil
	historyFile = ""
	msgMu.Unlock()
	t.Cleanup(func() {
		msgMu.Lock()
		messageRecords = nil
		historyFile = ""
		msgMu.Unlock()
	})
}

func TestAddMessageRecord_FifoTrim(t *testing.T) {
	resetRecords(t)
	for i := 0; i < maxRecords+1; i++ {
		addMessageRecord(MessageRecord{
			Type:      "group",
			SenderId:  1,
			Content:   strconv.Itoa(i),
			Timestamp: time.Now(),
		})
	}

	msgMu.Lock()
	defer msgMu.Unlock()
	if len(messageRecords) != maxRecords {
		t.Fatalf("超出上限后长度应恒为 %d, got %d", maxRecords, len(messageRecords))
	}
	// FIFO：最旧的 "0" 应被淘汰，队列头是 "1"、队尾是最新一条
	if first := messageRecords[0].Content; first != "1" {
		t.Errorf("最旧记录应被淘汰, 队头期望 \"1\", got %q", first)
	}
	if last := messageRecords[len(messageRecords)-1].Content; last != strconv.Itoa(maxRecords) {
		t.Errorf("队尾应是最新记录, 期望 %q, got %q", strconv.Itoa(maxRecords), last)
	}
}

// TestAddMessageRecord_Concurrent 并发写入，依赖 -race 检测器验证 msgMu 保护是否完备。
func TestAddMessageRecord_Concurrent(t *testing.T) {
	resetRecords(t)
	const goroutines = 8
	const perG = 200

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				addMessageRecord(MessageRecord{
					Type:      "group",
					SenderId:  1,
					Content:   strconv.Itoa(base + i),
					Timestamp: time.Now(),
				})
			}
		}(g * perG)
	}
	wg.Wait()

	msgMu.Lock()
	defer msgMu.Unlock()
	// 总写入量超过上限，最终长度必须恰好被裁剪到上限
	if len(messageRecords) != maxRecords {
		t.Fatalf("并发写入后长度应为 %d, got %d", maxRecords, len(messageRecords))
	}
}
