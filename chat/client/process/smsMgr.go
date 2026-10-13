package process

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/VKGQQ/goproject_1/chat/common/message"
)

// maxRecords 信息列表的最大容量。
// 只 append 不删会导致内存单调增长；达到上限后用 copy 前移覆盖最旧记录。
// 不能用 records = records[1:] 代替：切片只移动起始指针，底层数组不缩容，
// 前面的元素仍被底层数组引用，GC 不会回收，等于没删。
const maxRecords = 1000

// messageRecords 会被两个 goroutine 并发访问：
// 接收 goroutine（每收到一条消息写入）和菜单 goroutine（查看信息列表时读取）。
// 写多读少（每条消息都写，列表偶尔才看），因此选 Mutex 而非 RWMutex——
// RWMutex 的读路径有额外开销，读少写多场景下反而更慢。
var (
	messageRecords []MessageRecord
	msgMu          sync.Mutex
)

// MessageRecord 信息列表的统一记录格式：群聊、私聊、系统提示归一化为同一结构，
// 存结构体而非格式化字符串，保留 Type/SenderId 字段便于将来按类型、按发送人过滤。
type MessageRecord struct {
	Type      string // "group" / "private" / "system"
	SenderId  int
	Content   string
	Timestamp time.Time
}

// addMessageRecord 追加一条消息记录；容量满时 FIFO 淘汰最旧的一条。
func addMessageRecord(rec MessageRecord) {
	msgMu.Lock()
	defer msgMu.Unlock()
	if len(messageRecords) >= maxRecords {
		copy(messageRecords, messageRecords[1:]) // 整体前移一格，真正覆盖最旧记录
		messageRecords[maxRecords-1] = rec
		return
	}
	messageRecords = append(messageRecords, rec)
}

// outputMessageRecords 打印全部消息记录（菜单“信息列表”）。
// 数据量小（上限 1000 条），锁内直接打印即可，无需快照。
func outputMessageRecords() {
	msgMu.Lock()
	defer msgMu.Unlock()
	if len(messageRecords) == 0 {
		fmt.Println("暂无消息记录")
		fmt.Println()
		return
	}
	fmt.Println("----------消息记录----------")
	for _, rec := range messageRecords {
		t := rec.Timestamp.Format("15:04:05")
		switch rec.Type {
		case "group":
			fmt.Printf("[%s] 群聊 用户%d: %s\n", t, rec.SenderId, rec.Content)
		case "private":
			fmt.Printf("[%s] 私聊 用户%d: %s\n", t, rec.SenderId, rec.Content)
		case "system":
			fmt.Printf("[%s] 系统: %s\n", t, rec.Content)
		}
	}
	fmt.Println()
}

func outputGroupMes(mes *message.Message) {
	//反序列化mes.Data
	var smsMes message.SmsMes
	err := json.Unmarshal([]byte(mes.Data), &smsMes)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err.Error())
		return
	}

	// 写入信息列表（群聊）
	addMessageRecord(MessageRecord{
		Type:      "group",
		SenderId:  smsMes.UserId,
		Content:   smsMes.Content,
		Timestamp: time.Now(),
	})

	//显示信息
	info := fmt.Sprintf("接收到用户id:\t%d 发来的信息:\t%s",
		smsMes.UserId, smsMes.Content)
	fmt.Println(info)
	fmt.Println()
}

func outputPrivateMes(mes *message.Message) {
	var smsPrivateMes message.SmsPrivateMes
	err := json.Unmarshal([]byte(mes.Data), &smsPrivateMes)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err.Error())
		return
	}
	// UserId == 0 是服务端发来的系统提示（如"用户不在线"）
	if smsPrivateMes.UserId == 0 {
		addMessageRecord(MessageRecord{
			Type:      "system",
			Content:   smsPrivateMes.Content,
			Timestamp: time.Now(),
		})
		fmt.Printf("[系统信息] %s\n\n", smsPrivateMes.Content)
		return
	}
	// 写入信息列表（私聊）
	addMessageRecord(MessageRecord{
		Type:      "private",
		SenderId:  smsPrivateMes.UserId,
		Content:   smsPrivateMes.Content,
		Timestamp: time.Now(),
	})
	info := fmt.Sprintf("[私聊信息] 用户%d说：\t%s",
		smsPrivateMes.UserId, smsPrivateMes.Content)
	fmt.Println(info)
	fmt.Println()
}
