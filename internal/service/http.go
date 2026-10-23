package proxy

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"time"
)

func HandleHTTPRequest(clientConn net.Conn, hosts map[string]string) {
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		Logger().Warn("failed to read HTTP request", "err", err)
		return
	}
	defer req.Body.Close()

	hostname := req.URL.Hostname()
	port := DefaultPort(req.URL.Port(), "80")
	targetHost := net.JoinHostPort(hostname, port)
	hit := false

	if ip, ok := hosts[hostname]; ok {
		targetHost = net.JoinHostPort(ip, port)
		hit = true
		Logger().Info("host resolved", "domain", hostname, "ip", ip)
	}

	serverConn, err := net.DialTimeout("tcp", targetHost, 30*time.Second)
	if err != nil {
		Logger().Warn("upstream unreachable", "target", targetHost, "err", err)
		WriteProxyError(clientConn, http.StatusBadGateway, err.Error())
		return
	}
	defer serverConn.Close()

	req.RequestURI = ""
	req.URL.Scheme = ""
	req.URL.Host = ""
	req.Host = hostname
	if port != "80" {
		req.Host = net.JoinHostPort(hostname, port)
	}

	req = req.WithContext(ctx)

	if err := req.Write(serverConn); err != nil {
		Logger().Warn("failed to forward request", "err", err)
		WriteProxyError(clientConn, http.StatusBadGateway, err.Error())
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(serverConn), req)
	if err != nil {
		Logger().Warn("failed to read upstream response", "err", err)
		WriteProxyError(clientConn, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()

	if err := resp.Write(clientConn); err != nil {
		Logger().Warn("failed to write response to client", "err", err)
		return
	}

	if !hit {
		Logger().Debug("request proxied", "host", hostname, "port", port)
	}
}
