package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func startEchoServer(t *testing.T) (addr string, port string) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen echo server: %v", err)
	}
	t.Cleanup(func() { l.Close() })

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		io.Copy(conn, conn)
	}()

	_, port, err = net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatalf("split echo address: %v", err)
	}
	return l.Addr().String(), port
}

func TestHandlerConnectTunnel(t *testing.T) {
	_, upstreamPort := startEchoServer(t)
	proxySrv := startProxyHandler(t, map[string]string{"example.com": "127.0.0.1"})
	conn, br := dialProxy(t, proxySrv)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	fmt.Fprintf(conn, "CONNECT example.com:%s HTTP/1.1\r\nHost: example.com:%s\r\n\r\n", upstreamPort, upstreamPort)
	if code := readResponseStatus(t, br); code != http.StatusOK {
		t.Fatalf("CONNECT response: got %d, want 200", code)
	}

	msg := []byte("hello through tunnel")
	if _, err := conn.Write(msg); err != nil {
		t.Fatalf("write tunnel data: %v", err)
	}
	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(br, buf); err != nil {
		t.Fatalf("read tunnel data: %v", err)
	}
	if string(buf) != string(msg) {
		t.Fatalf("tunnel echo: got %q, want %q", buf, msg)
	}
}

func TestHandlerConnectPreservesBufferedData(t *testing.T) {
	_, upstreamPort := startEchoServer(t)
	proxySrv := startProxyHandler(t, map[string]string{"example.com": "127.0.0.1"})
	conn, br := dialProxy(t, proxySrv)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	payload := "early-data"
	request := fmt.Sprintf("CONNECT example.com:%s HTTP/1.1\r\nHost: example.com:%s\r\nProxy-Connection: keep-alive\r\n\r\n%s", upstreamPort, upstreamPort, payload)
	if _, err := io.WriteString(conn, request); err != nil {
		t.Fatalf("write CONNECT with early data: %v", err)
	}

	if code := readResponseStatus(t, br); code != http.StatusOK {
		t.Fatalf("CONNECT response: got %d, want 200", code)
	}
	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(br, buf); err != nil {
		t.Fatalf("read early data: %v", err)
	}
	if string(buf) != payload {
		t.Fatalf("early data: got %q, want %q", buf, payload)
	}
}

func TestHandlerConnectUnreachable(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	_, closedPort, _ := net.SplitHostPort(l.Addr().String())
	l.Close()

	proxySrv := startProxyHandler(t, map[string]string{"example.com": "127.0.0.1"})
	conn, br := dialProxy(t, proxySrv)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	fmt.Fprintf(conn, "CONNECT example.com:%s HTTP/1.1\r\nHost: example.com:%s\r\n\r\n", closedPort, closedPort)
	if code := readResponseStatus(t, br); code != http.StatusBadGateway {
		t.Fatalf("CONNECT unreachable: got %d, want 502", code)
	}
}

func TestHandlerConnectMissingHost(t *testing.T) {
	proxySrv := startProxyHandler(t, nil)
	conn, br := dialProxy(t, proxySrv)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	fmt.Fprint(conn, "CONNECT :443 HTTP/1.1\r\nHost: :443\r\n\r\n")
	if code := readResponseStatus(t, br); code != http.StatusBadRequest {
		t.Fatalf("CONNECT missing host: got %d, want 400", code)
	}
}

func TestParseConnectTarget(t *testing.T) {
	tests := []struct {
		in       string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{"example.com:8443", "example.com", "8443", false},
		{"example.com", "example.com", "443", false},
		{"example.com:", "example.com", "443", false},
		{"[::1]:8443", "::1", "8443", false},
		{"[::1]", "::1", "443", false},
		{"::1", "", "", true},
		{"", "", "", true},
	}
	for _, tc := range tests {
		host, port, err := parseConnectTarget(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseConnectTarget(%q) expected error, got host=%q port=%q", tc.in, host, port)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseConnectTarget(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if host != tc.wantHost || port != tc.wantPort {
			t.Errorf("parseConnectTarget(%q) = (%q, %q), want (%q, %q)", tc.in, host, port, tc.wantHost, tc.wantPort)
		}
	}
}
