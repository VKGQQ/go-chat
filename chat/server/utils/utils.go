// Package utils 提供服务端的协议帧读写工具。
package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/VKGQQ/go-chat/chat/common/message"
)

// MaxPkgSize 是协议允许的单个包体最大字节数。
// 超过该长度的包视为畸形包/攻击包，直接拒绝，避免切片越界。
const MaxPkgSize = 8096

// Transfer 协议帧读写器：按“4 字节大端长度 + JSON 包体”的格式收发消息。
type Transfer struct {
	Conn net.Conn
	Buf  [8096]byte
}

// ReadPkg 按“4字节大端长度 + JSON包体”的协议格式读取一个完整的包。
func (this *Transfer) ReadPkg() (mes message.Message, err error) {
	// 读帧头：TCP 流不保证一次 Read 读满，必须用 io.ReadFull 读满 4 字节，
	// 否则只读到部分长度字节会算出错误长度，导致后续所有帧错位。
	if _, err = io.ReadFull(this.Conn, this.Buf[:4]); err != nil {
		return
	}
	pkgLen := binary.BigEndian.Uint32(this.Buf[:4])
	// 长度校验：长度字段来自网络，不可信，必须先验货再使用
	if pkgLen == 0 || pkgLen > MaxPkgSize {
		err = fmt.Errorf("非法包长度: %d", pkgLen)
		return
	}
	// 按实际长度动态分配缓冲区并读满整个包体，短读时 io.ReadFull 会自动补齐
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

// WritePkg 将 data 封装为一个完整协议帧写出（4 字节长度头 + 包体）。
func (this *Transfer) WritePkg(data []byte) (err error) {
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
