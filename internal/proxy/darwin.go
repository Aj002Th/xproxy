//go:build darwin

package proxy

import (
	"net"
)

// 获取原始目标地址（macOS 不支持，直接返回错误）
func getOriginalDst(conn *net.TCPConn) (string, error) {
	return "", fmt.Errorf("macOS does not support transparent proxy")
}
