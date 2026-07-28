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

	if msg != "" {
		resp.Body = &errorBody{msg: msg}
		resp.ContentLength = int64(len(msg))
	} else {
		resp.Body = http.NoBody
		resp.ContentLength = 0
	}

	if err := resp.Write(conn); err != nil {
		Logger.Warn("failed to write proxy error response", "err", err)
	}
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
