// Package utils 提供客户端的协议帧读写工具和标准输入读取工具。
package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/VKGQQ/goproject_1/chat/common/message"
)

// MaxPkgSize 是协议允许的单个包体最大字节数。
const MaxPkgSize = 8096

// connWriteTimeout 单次发包（含帧头和帧体）的写超时时间。
// 与服务端保持一致：防止对端或网络卡住时 Conn.Write 永久阻塞，
// 否则菜单发送会冻死整个客户端。
const connWriteTimeout = 5 * time.Second

// writeMu 保护客户端到服务端的唯一一条连接。
// 目前只有菜单 goroutine 一个写者，但后续新增心跳、私聊等并发写者时，
// 帧头+帧体必须持锁连续写出，否则字节交错导致对端帧失步。
var writeMu sync.Mutex

// Transfer 协议帧读写器：按“4 字节大端长度 + JSON 包体”的格式收发消息。
type Transfer struct {
	Conn net.Conn
	Buf  [8096]byte
}

// ReadPkg 按“4字节大端长度 + JSON包体”的协议格式读取一个完整的包。
func (this *Transfer) ReadPkg() (mes message.Message, err error) {
	// 读帧头：用 io.ReadFull 读满 4 字节，防止 TCP 短读导致长度解析错误
	if _, err = io.ReadFull(this.Conn, this.Buf[:4]); err != nil {
		return
	}
	pkgLen := binary.BigEndian.Uint32(this.Buf[:4])
	// 长度校验：长度字段来自网络，不可信
	if pkgLen == 0 || pkgLen > MaxPkgSize {
		err = fmt.Errorf("非法包长度: %d", pkgLen)
		return
	}
	// 按实际长度动态分配缓冲区并读满整个包体
	body := make([]byte, pkgLen)
	if _, err = io.ReadFull(this.Conn, body); err != nil {
		return
	}
	if err = json.Unmarshal(body, &mes); err != nil {
		fmt.Println("json.Unmarshal err:", err)
		return
	}
	return
}

// WritePkg 向连接写入一个完整协议帧：4 字节大端长度 + JSON 包体。
// 持写锁保证多个 goroutine 并发发包时帧不错乱；
// 每次发包前设置写截止时间，防止对端卡死时永久阻塞。
func (this *Transfer) WritePkg(data []byte) (err error) {
	writeMu.Lock()
	defer writeMu.Unlock()
	if err = this.Conn.SetWriteDeadline(time.Now().Add(connWriteTimeout)); err != nil {
		return
	}
	var pkgLen uint32
	pkgLen = uint32(len(data))
	binary.BigEndian.PutUint32(this.Buf[0:4], pkgLen)
	n, err := this.Conn.Write(this.Buf[:4])
	if n != 4 || err != nil {
		fmt.Println("conn.Write err:", err)
		return
	}
	n, err = this.Conn.Write(data)
	if n != int(pkgLen) || err != nil {
		fmt.Println("conn.Write err:", err)
		return
	}
	return
}
