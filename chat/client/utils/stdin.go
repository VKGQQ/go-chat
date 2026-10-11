package utils

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// stdinReader 是整个客户端唯一的标准输入缓冲读取器。
// 所有包都必须通过 ReadLine/ReadInt 读取输入，禁止再使用 fmt.Scan/Scanln
// 直接读 os.Stdin：bufio 会预读数据到自己的缓冲区，混用会导致未走缓冲区的
// 读取方永远拿不到被预读扣留下的内容。
var stdinReader = bufio.NewReader(os.Stdin)

// ReadLine 读取一整行输入。
// 以换行作为分隔符，因此行内空格会被完整保留（消息、昵称可包含空格）；
// 返回前去掉首尾空白（含 Windows 行尾的 \r\n）。
// 若读到 EOF 且没有任何残余内容，返回空字符串和错误。
func ReadLine() (string, error) {
	line, err := stdinReader.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil && line == "" {
		return "", err
	}
	return line, nil
}

// ReadInt 读取一整行并解析为整数。
// 输入不是合法数字时返回错误，由调用方提示并重新选择菜单。
func ReadInt() (int, error) {
	line, err := ReadLine()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(line)
}
