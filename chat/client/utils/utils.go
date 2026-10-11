package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/VKGQQ/goproject_1/chat/common/message"
)

// MaxPkgSize 是协议允许的单个包体最大字节数。
const MaxPkgSize = 8096

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
