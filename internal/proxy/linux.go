//go:build linux

package proxy

import (
	"net"
	"syscall"
)

// 获取原始目标地址（仅适用于 Linux）
func getOriginalDst(conn *net.TCPConn) (string, error) {
	file, err := conn.File()
	if err != nil {
		return "", err
	}
	defer file.Close()

	addr, err := syscall.GetsockoptIPv6Mreq(int(file.Fd()), syscall.IPPROTO_IP, syscall.SO_ORIGINAL_DST)
	if err != nil {
		return "", err
	}

	ip := net.IPv4(byte(addr.Multiaddr[4]), byte(addr.Multiaddr[5]), byte(addr.Multiaddr[6]), byte(addr.Multiaddr[7]))
	port := uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])
	return fmt.Sprintf("%s:%d", ip.String(), port), nil
}
