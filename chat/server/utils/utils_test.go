package utils

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

// pipeDeadline 测试护栏：net.Pipe 是全双工阻塞管道，
// 一旦被测代码死锁，测试应在超时后失败而不是永久挂起。
const pipeDeadline = 2 * time.Second

func TestWritePkgReadPkg_RoundTrip(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	_ = server.SetDeadline(time.Now().Add(pipeDeadline))
	_ = client.SetDeadline(time.Now().Add(pipeDeadline))

	go func() {
		_ = (&Transfer{Conn: client}).WritePkg([]byte(`{"type":"LoginMes","data":"{\"userId\":100}"}`))
	}()

	mes, err := (&Transfer{Conn: server}).ReadPkg()
	if err != nil {
		t.Fatalf("ReadPkg 失败: %v", err)
	}
	if mes.Type != "LoginMes" || mes.Data != `{"userId":100}` {
		t.Errorf("帧内容回环不一致: type=%s data=%s", mes.Type, mes.Data)
	}
}

func TestReadPkg_RejectsInvalidLength(t *testing.T) {
	tests := []struct {
		name   string
		pkgLen uint32
	}{
		{"零长度包", 0},
		{"超过上限的包", MaxPkgSize + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()
			_ = server.SetDeadline(time.Now().Add(pipeDeadline))
			_ = client.SetDeadline(time.Now().Add(pipeDeadline))

			var header [4]byte
			binary.BigEndian.PutUint32(header[:], tt.pkgLen)
			go func() { _, _ = client.Write(header[:]) }()

			if _, err := (&Transfer{Conn: server}).ReadPkg(); err == nil {
				t.Fatalf("长度 %d 应被拒绝", tt.pkgLen)
			}
		})
	}
}

// TestReadPkg_ReassemblesFragmentedFrame 模拟 TCP 半包：
// 一帧分两次写入，io.ReadFull 必须读满拼出完整帧。
func TestReadPkg_ReassemblesFragmentedFrame(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	_ = server.SetDeadline(time.Now().Add(pipeDeadline))
	_ = client.SetDeadline(time.Now().Add(pipeDeadline))

	body := []byte(`{"type":"SmsMes","data":"hello"}`)
	frame := make([]byte, 4+len(body))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(body)))
	copy(frame[4:], body)

	go func() {
		_, _ = client.Write(frame[:6]) // 帧头 + 2 字节包体
		time.Sleep(50 * time.Millisecond)
		_, _ = client.Write(frame[6:])
	}()

	mes, err := (&Transfer{Conn: server}).ReadPkg()
	if err != nil {
		t.Fatalf("半包场景 ReadPkg 失败: %v", err)
	}
	if mes.Type != "SmsMes" || mes.Data != "hello" {
		t.Errorf("重组后的帧内容不一致: type=%s data=%s", mes.Type, mes.Data)
	}
}

// TestReadPkg_PeerClosed 锁定 serve() 清理路径的前提：
// 对端正常关闭时 ReadPkg 必须返回 io.EOF。
func TestReadPkg_PeerClosed(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	_ = client.Close()

	if _, err := (&Transfer{Conn: server}).ReadPkg(); !errors.Is(err, io.EOF) {
		t.Fatalf("对端关闭应返回 io.EOF, got %v", err)
	}
}
