package proxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeHostsFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write hosts file: %v", err)
	}
	return path
}

func TestStartProxyReturnsHostsError(t *testing.T) {
	if err := StartProxy(context.Background(), "/nonexistent/hosts", "127.0.0.1:0"); err == nil {
		t.Fatal("expected an error for a missing hosts file")
	}
}

func TestStartProxyReturnsBindError(t *testing.T) {
	path := writeHostsFile(t, "127.0.0.1 localhost\n")
	if err := StartProxy(context.Background(), path, "invalid:addr:99999"); err == nil {
		t.Fatal("expected an error for an invalid listen address")
	}
}

func TestServeHTTPEndToEnd(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from upstream"))
	}))
	defer upstream.Close()

	_, upstreamPort, _ := net.SplitHostPort(upstream.Listener.Addr().String())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- serve(ctx, listener, map[string]string{"example.com": "127.0.0.1"})
	}()

	proxyURL, err := url.Parse("http://" + listener.Addr().String())
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   5 * time.Second,
	}

	resp, err := client.Get("http://example.com:" + upstreamPort + "/")
	if err != nil {
		cancel()
		t.Fatalf("GET through proxy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello from upstream" {
		t.Errorf("body: got %q, want hello from upstream", body)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not stop after context cancellation")
	}
}

func TestServeReturnsErrorOnClosedListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	listener.Close()

	if err := serve(context.Background(), listener, nil); err == nil {
		t.Fatal("expected an error from a closed listener")
	}
}

func TestServeGracefulShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- serve(ctx, listener, nil)
	}()

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not stop after context cancellation")
	}
}
