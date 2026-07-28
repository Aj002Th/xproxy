package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStartProxyHTTPEndToEnd(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "example.com" {
			t.Errorf("Host header: got %q, want example.com", r.Host)
		}
		w.WriteHeader(200)
		w.Write([]byte("hello from upstream"))
	}))
	defer upstream.Close()

	dir := t.TempDir()
	hostsPath := filepath.Join(dir, "hosts")
	os.WriteFile(hostsPath, []byte("127.0.0.1 example.com\n"), 0644)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get ephemeral port: %v", err)
	}
	listenAddr := l.Addr().String()
	l.Close()
	go StartProxy(hostsPath, listenAddr)
	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", listenAddr)
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	upstreamPort := upstream.Listener.Addr().(*net.TCPAddr).Port
	req := fmt.Sprintf("GET http://example.com:%d/ HTTP/1.1\r\nHost: example.com\r\n\r\n", upstreamPort)
	conn.Write([]byte(req))

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello from upstream" {
		t.Errorf("body: got %q, want hello from upstream", string(body))
	}
}

func TestStartProxyHostsNotFound(t *testing.T) {
	StartProxy("/nonexistent/hosts", "127.0.0.1:0")
}

func TestStartProxyBindFailure(t *testing.T) {
	StartProxy("/etc/hosts", "invalid:addr:99999")
}

func TestHandleClientConnectDispatches(t *testing.T) {
	upstream, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start upstream: %v", err)
	}
	defer upstream.Close()

	go func() {
		conn, err := upstream.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		io.Copy(conn, conn)
	}()

	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	hosts := map[string]string{"example.com": "127.0.0.1"}
	upstreamHost := upstream.Addr().String()

	go handleClient(serverConn, hosts)
	clientConn.Write([]byte(fmt.Sprintf("CONNECT %s HTTP/1.1\r\n\r\n", upstreamHost)))

	resp, err := http.ReadResponse(bufio.NewReader(clientConn), nil)
	if err != nil {
		t.Fatalf("failed to read CONNECT response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 Connection Established, got %d", resp.StatusCode)
	}
	clientConn.Close()
}
