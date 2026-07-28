package proxy

import (
	"io"
	"net"
)

func HandleHTTPSRequest(clientConn net.Conn, hosts map[string]string, targetHost string) {
	defer clientConn.Close()

	host, port, err := net.SplitHostPort(targetHost)
	if err != nil {
		Logger.Warn("invalid CONNECT target", "target", targetHost, "err", err)
		return
	}

	port = DefaultPort(port, "443")
	originalHost := host
	hit := false

	if ip, ok := hosts[host]; ok {
		host = ip
		hit = true
		Logger.Info("host resolved", "domain", originalHost, "ip", ip)
	}

	serverConn, err := net.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		Logger.Warn("upstream unreachable", "target", net.JoinHostPort(host, port), "err", err)
		WriteProxyError(clientConn, 502, err.Error())
		return
	}
	defer serverConn.Close()

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	if !hit {
		Logger.Debug("tunnel established", "host", host, "port", port)
	}

	go func() {
		defer serverConn.Close()
		defer clientConn.Close()
		io.Copy(serverConn, clientConn)
	}()
	io.Copy(clientConn, serverConn)
}

type rewindReader struct {
	prefix    string
	backend   io.Reader
	prefixOff int
}

func (r *rewindReader) Read(p []byte) (int, error) {
	if r.prefixOff < len(r.prefix) {
		n := copy(p, r.prefix[r.prefixOff:])
		r.prefixOff += n
		return n, nil
	}
	return r.backend.Read(p)
}

func RewindReader(r io.Reader, prefix string) io.Reader {
	return &rewindReader{prefix: prefix, backend: r}
}
