package proxy

import (
	"io"
	"net"
	"net/http"
)

func DefaultPort(port string, defaultPort string) string {
	if port == "" {
		return defaultPort
	}
	return port
}

func WriteProxyError(conn net.Conn, statusCode int, msg string) {
	resp := &http.Response{
		StatusCode: statusCode,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "text/plain")
	resp.Header.Set("Connection", "close")
	resp.Body = http.NoBody
	resp.ContentLength = 0

	if msg != "" {
		resp.Header.Set("X-Proxy-Error", msg)
		resp.Body = &errorBody{msg: msg}
		resp.ContentLength = int64(len(msg))
	}
	resp.Write(conn)
}

type errorBody struct {
	msg string
	pos int
}

func (b *errorBody) Read(p []byte) (int, error) {
	if b.pos >= len(b.msg) {
		return 0, io.EOF
	}
	n := copy(p, b.msg[b.pos:])
	b.pos += n
	return n, nil
}

func (b *errorBody) Close() error {
	return nil
}
