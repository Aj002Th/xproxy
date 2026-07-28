package proxy

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestDefaultPort(t *testing.T) {
	tests := []struct {
		host, port, def, want string
	}{
		{"example.com", "", "80", "80"},
		{"example.com", "8080", "80", "8080"},
		{"example.com", "443", "80", "443"},
	}
	for _, tc := range tests {
		got := DefaultPort(tc.port, tc.def)
		if got != tc.want {
			t.Errorf("DefaultPort(%q, %q) = %q, want %q", tc.port, tc.def, got, tc.want)
		}
	}
}

func TestWriteProxyError(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		WriteProxyError(server, http.StatusBadGateway, "upstream refused connection")
		server.Close()
	}()

	resp, err := http.ReadResponse(bufio.NewReader(client), nil)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
	if !strings.Contains(resp.Status, "Bad Gateway") {
		t.Errorf("status text = %q, want Bad Gateway", resp.Status)
	}
}

func TestWriteProxyErrorEmptyMsg(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		WriteProxyError(server, http.StatusServiceUnavailable, "")
		server.Close()
	}()

	resp, err := http.ReadResponse(bufio.NewReader(client), nil)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
	if resp.ContentLength != 0 {
		t.Errorf("empty msg should have content-length 0, got %d", resp.ContentLength)
	}
}
