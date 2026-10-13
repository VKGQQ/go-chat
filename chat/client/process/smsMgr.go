package process

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	// historyFile 当前登录用户的历史文件路径，登录成功后由 loadHistory 设置
	historyFile string
)

// historyDir 聊天记录的专用目录（相对客户端启动目录）
const historyDir = "history"

// historyFilePath 按用户区分历史文件，同一台机器登录不同账号互不串台
func historyFilePath(userId int) string {
	return filepath.Join(historyDir, fmt.Sprintf("chat_history_%d.json", userId))
}

// MessageRecord 信息列表的统一记录格式：群聊、私聊、系统提示归一化为同一结构，
// 存结构体而非格式化字符串，保留 Type/SenderId 字段便于将来按类型、按发送人过滤。
type MessageRecord struct {
	Type      string // "group" / "private" / "system"
	SenderId  int
	Content   string
	Timestamp time.Time
}

// addMessageRecord 追加一条消息记录；容量满时 FIFO 淘汰最旧的一条。
// 内存与文件是同一条消息的两个副本，写入必须在同一把锁内完成：
// 锁外写文件会因 goroutine 调度抖动导致两个副本的记录顺序错乱。
func addMessageRecord(rec MessageRecord) {
	msgMu.Lock()
	defer msgMu.Unlock()
	if len(messageRecords) >= maxRecords {
		copy(messageRecords, messageRecords[1:]) // 整体前移一格，真正覆盖最旧记录
		messageRecords[maxRecords-1] = rec
	} else {
		messageRecords = append(messageRecords, rec)
	}
	appendHistoryFile(rec)
}

// appendHistoryFile 把一条记录追加到历史文件（JSONL 格式，一行一条）。
// JSONL 支持 O_APPEND 直接追加，无需读旧内容整体重写；
// 且进程崩溃最多截断最后一行，加载时逐行解析、坏行跳过，故障隔离在行级别。
// 每条消息即时落盘（WAL 思想），不依赖“正常退出时保存”，也无需拦截退出信号。
// 调用方必须持有 msgMu。
func appendHistoryFile(rec MessageRecord) {
	if historyFile == "" {
		return // 未登录时不会有消息到达，防御性判断
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("打开历史文件失败:", err)
		return
	}
	defer func() { _ = f.Close() }()
	if _, err = f.Write(append(data, '\n')); err != nil {
		fmt.Println("写入历史文件失败:", err)
	}
}

// loadHistory 登录成功后加载该用户的本地聊天记录到内存。
// 文件不存在属正常情况（首次使用），视为空历史。
func loadHistory(userId int) {
	// 首次运行时目录不存在，提前创建；O_APPEND 不会自动创建父目录
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		fmt.Println("创建历史目录失败:", err)
		return
	}
	historyFile = historyFilePath(userId)
	data, err := os.ReadFile(historyFile)
	if err != nil {
		return
	}
	msgMu.Lock()
	defer msgMu.Unlock()
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var rec MessageRecord
		if json.Unmarshal([]byte(line), &rec) != nil {
			continue // 崩溃截断的坏行：跳过该行，不影响其他记录
		}
		messageRecords = append(messageRecords, rec)
	}
	// 文件可能比内存上限大，只保留最近 maxRecords 条；
	// 拷贝到新 slice，避免底层数组一直引用被丢弃的旧记录
	if len(messageRecords) > maxRecords {
		trimmed := make([]MessageRecord, maxRecords)
		copy(trimmed, messageRecords[len(messageRecords)-maxRecords:])
		messageRecords = trimmed
	}
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

// outputGroupMes 处理服务端下行的群聊消息：记录到信息列表并打印。
func outputGroupMes(mes *message.Message) {
	var smsMes message.SmsMes
	err := json.Unmarshal([]byte(mes.Data), &smsMes)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err.Error())
		return
	}

	addMessageRecord(MessageRecord{
		Type:      "group",
		SenderId:  smsMes.UserId,
		Content:   smsMes.Content,
		Timestamp: time.Now(),
	})

	info := fmt.Sprintf("接收到用户id:\t%d 发来的信息:\t%s",
		smsMes.UserId, smsMes.Content)
	fmt.Println(info)
	fmt.Println()
}

// outputPrivateMes 处理服务端下行的私聊消息：记录到信息列表并打印。
// UserId == 0 的消息视为系统提示（如“用户不在线”），单独归类展示。
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
