//go:build windows

package proxy

import (
	"fmt"
	"net"
)

// 获取原始目标地址（Windows 不支持，直接返回错误）
func getOriginalDst(conn *net.TCPConn) (string, error) {
	return "", fmt.Errorf("Windows does not support transparent proxy")
}
