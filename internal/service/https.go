package proxy

import (
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func HandleHTTPSRequest(clientConn net.Conn, hosts map[string]string, targetHost string) {
	defer clientConn.Close()

	if !strings.Contains(targetHost, ":") {
		targetHost += ":443"
	}

	host, port, err := net.SplitHostPort(targetHost)
	if err != nil {
		Logger().Warn("invalid CONNECT target", "target", targetHost, "err", err)
		return
	}

	port = DefaultPort(port, "443")
	originalHost := host
	hit := false

	if ip, ok := hosts[host]; ok {
		host = ip
		hit = true
		Logger().Info("host resolved", "domain", originalHost, "ip", ip)
	}

	serverConn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 30*time.Second)
	if err != nil {
		Logger().Warn("upstream unreachable", "target", net.JoinHostPort(host, port), "err", err)
		WriteProxyError(clientConn, http.StatusBadGateway, err.Error())
		return
	}
	defer serverConn.Close()

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		Logger().Warn("failed to send CONNECT response", "err", err)
		return
	}

	if !hit {
		Logger().Debug("tunnel established", "host", host, "port", port)
	}

	go func() {
		defer serverConn.Close()
		defer clientConn.Close()
		io.Copy(serverConn, clientConn)
	}()
	io.Copy(clientConn, serverConn)
}
