package proxy

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func init() {
	Logger = slog.Default()
}

func TestHandleHTTPRequestProxies(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	reqStr := "GET http://127.0.0.1:9999/foo HTTP/1.1\r\nHost: 127.0.0.1:9999\r\n\r\n"
	hosts := map[string]string{}

	go HandleHTTPRequest(clientConn, hosts)
	serverConn.Write([]byte(reqStr))

	resp, err := http.ReadResponse(bufio.NewReader(serverConn), nil)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 502 {
		t.Errorf("expected 502 for unresolvable host, got %d", resp.StatusCode)
	}
}

func TestHandleHTTPRequestHostResolution(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "example.com" {
			t.Errorf("upstream Host header: got %q, want example.com", r.Host)
		}
		if r.URL.Path != "/path" {
			t.Errorf("upstream path: got %q, want /path", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	backendHost := strings.TrimPrefix(backend.URL, "http://")
	backendIP := strings.Split(backendHost, ":")[0]
	backendPort := strings.Split(backendHost, ":")[1]
	hosts := map[string]string{"example.com": backendIP}

	clientConn, serverConn := net.Pipe()

	req := fmt.Sprintf("GET http://example.com:%s/path HTTP/1.1\r\nHost: example.com\r\n\r\n", backendPort)

	go HandleHTTPRequest(clientConn, hosts)
	serverConn.Write([]byte(req))

	resp, err := http.ReadResponse(bufio.NewReader(serverConn), nil)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("body: got %q, want ok", string(body))
	}
	serverConn.Close()
}
