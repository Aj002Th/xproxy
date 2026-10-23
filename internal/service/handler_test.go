package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func startProxyHandler(t *testing.T, hosts map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewHandler(hosts))
	t.Cleanup(srv.Close)
	return srv
}

func proxyClient(t *testing.T, proxySrv *httptest.Server) *http.Client {
	t.Helper()
	u, err := url.Parse(proxySrv.URL)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}
	return &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(u)},
		Timeout:   5 * time.Second,
	}
}

func dialProxy(t *testing.T, proxySrv *httptest.Server) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.Dial("tcp", proxySrv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn, bufio.NewReader(conn)
}

func backendPort(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split backend address: %v", err)
	}
	return port
}

// readResponseStatus reads one response line plus headers and returns the
// status code. It is used for CONNECT/101 where the response has no body.
func readResponseStatus(t *testing.T, br *bufio.Reader) int {
	t.Helper()
	line, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read status line: %v", err)
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		t.Fatalf("malformed status line %q", line)
	}
	code, err := strconv.Atoi(fields[1])
	if err != nil {
		t.Fatalf("parse status code %q: %v", fields[1], err)
	}
	for {
		line, err = br.ReadString('\n')
		if err != nil {
			t.Fatalf("read response header: %v", err)
		}
		if line == "\r\n" || line == "\n" {
			return code
		}
	}
}

func TestHandlerProxiesHTTP(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/foo" {
			t.Errorf("upstream path: got %q, want /foo", r.URL.Path)
		}
		if !strings.HasPrefix(r.Host, "example.com:") {
			t.Errorf("upstream Host: got %q, want example.com:<port>", r.Host)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	port := backendPort(t, backend)
	proxySrv := startProxyHandler(t, map[string]string{"example.com": "127.0.0.1"})
	client := proxyClient(t, proxySrv)

	resp, err := client.Get("http://example.com:" + port + "/foo")
	if err != nil {
		t.Fatalf("GET through proxy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("body: got %q, want ok", body)
	}
}

func TestHandlerWithoutHostsHit(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("proxied"))
	}))
	defer backend.Close()

	proxySrv := startProxyHandler(t, nil)
	client := proxyClient(t, proxySrv)

	resp, err := client.Get("http://" + backend.Listener.Addr().String() + "/")
	if err != nil {
		t.Fatalf("GET through proxy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "proxied" {
		t.Errorf("body: got %q, want proxied", body)
	}
}

func TestHandlerKeepsConnectionAlive(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	port := backendPort(t, backend)
	proxySrv := startProxyHandler(t, map[string]string{"example.com": "127.0.0.1"})
	conn, br := dialProxy(t, proxySrv)

	for i := 0; i < 2; i++ {
		fmt.Fprintf(conn, "GET http://example.com:%s/ HTTP/1.1\r\nHost: example.com:%s\r\n\r\n", port, port)
		resp, err := http.ReadResponse(br, nil)
		if err != nil {
			t.Fatalf("request %d: read response: %v", i+1, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != "ok" {
			t.Fatalf("request %d: got status=%d body=%q", i+1, resp.StatusCode, body)
		}
		if resp.Close {
			t.Fatalf("request %d: proxy closed a keep-alive connection", i+1)
		}
	}
}

func TestHandlerStripsHopByHopHeaders(t *testing.T) {
	got := make(chan http.Header, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	port := backendPort(t, backend)
	proxySrv := startProxyHandler(t, map[string]string{"example.com": "127.0.0.1"})
	conn, br := dialProxy(t, proxySrv)

	fmt.Fprintf(conn, "GET http://example.com:%s/ HTTP/1.1\r\nHost: example.com:%s\r\nProxy-Connection: keep-alive\r\nConnection: X-Drop\r\nX-Drop: secret\r\n\r\n", port, port)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	resp.Body.Close()

	headers := <-got
	for _, name := range []string{"Proxy-Connection", "X-Drop", "Connection"} {
		if value := headers.Get(name); value != "" {
			t.Errorf("upstream received hop-by-hop header %s=%q", name, value)
		}
	}
}

func TestHandlerForwardsInterimResponse(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen backend: %v", err)
	}
	defer backend.Close()

	go func() {
		conn, err := backend.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		br := bufio.NewReader(conn)
		if _, err := http.ReadRequest(br); err != nil {
			return
		}
		io.WriteString(conn, "HTTP/1.1 103 Early Hints\r\nLink: </style.css>; rel=preload\r\n\r\n")
		io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nok")
	}()

	_, port, _ := net.SplitHostPort(backend.Addr().String())
	proxySrv := startProxyHandler(t, map[string]string{"example.com": "127.0.0.1"})
	conn, br := dialProxy(t, proxySrv)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	fmt.Fprintf(conn, "GET http://example.com:%s/ HTTP/1.1\r\nHost: example.com:%s\r\nConnection: close\r\n\r\n", port, port)

	resp1, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read 103: %v", err)
	}
	if resp1.StatusCode != http.StatusEarlyHints {
		t.Fatalf("first response: got %d, want 103", resp1.StatusCode)
	}
	resp1.Body.Close()

	resp2, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read 200: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second response: got %d, want 200", resp2.StatusCode)
	}
	body, _ := io.ReadAll(resp2.Body)
	if string(body) != "ok" {
		t.Fatalf("body: got %q, want ok", body)
	}
}

func TestHandlerExpectContinue(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen backend: %v", err)
	}
	defer backend.Close()

	go func() {
		conn, err := backend.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		br := bufio.NewReader(conn)
		req, err := http.ReadRequest(br)
		if err != nil {
			return
		}
		io.WriteString(conn, "HTTP/1.1 100 Continue\r\n\r\n")
		body, _ := io.ReadAll(req.Body)
		req.Body.Close()
		fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body)
	}()

	_, port, _ := net.SplitHostPort(backend.Addr().String())
	proxySrv := startProxyHandler(t, map[string]string{"example.com": "127.0.0.1"})
	conn, br := dialProxy(t, proxySrv)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	fmt.Fprintf(conn, "POST http://example.com:%s/ HTTP/1.1\r\nHost: example.com:%s\r\nExpect: 100-continue\r\nContent-Length: 7\r\nConnection: close\r\n\r\n", port, port)

	if code := readResponseStatus(t, br); code != http.StatusContinue {
		t.Fatalf("first response: got %d, want 100", code)
	}
	if _, err := io.WriteString(conn, "payload"); err != nil {
		t.Fatalf("write body: %v", err)
	}

	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read final response: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("final status: got %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "payload" {
		t.Fatalf("body: got %q, want payload", body)
	}
}

func TestHandlerUpgradeTunnel(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Header.Get("Upgrade"), "echo") {
			t.Errorf("upstream Upgrade = %q, want echo", r.Header.Get("Upgrade"))
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("upstream does not support hijacking")
			return
		}
		conn, rw, err := hj.Hijack()
		if err != nil {
			t.Errorf("hijack upstream: %v", err)
			return
		}
		defer conn.Close()
		io.WriteString(rw, "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: echo\r\n\r\n")
		rw.Flush()
		io.Copy(conn, rw.Reader)
	}))
	defer backend.Close()

	port := backendPort(t, backend)
	proxySrv := startProxyHandler(t, map[string]string{"example.com": "127.0.0.1"})
	conn, br := dialProxy(t, proxySrv)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	fmt.Fprintf(conn, "GET http://example.com:%s/ HTTP/1.1\r\nHost: example.com:%s\r\nConnection: Upgrade\r\nUpgrade: echo\r\n\r\n", port, port)
	if code := readResponseStatus(t, br); code != http.StatusSwitchingProtocols {
		t.Fatalf("upgrade response: got %d, want 101", code)
	}

	if _, err := io.WriteString(conn, "ping"); err != nil {
		t.Fatalf("write tunnel data: %v", err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(br, buf); err != nil {
		t.Fatalf("read tunnel data: %v", err)
	}
	if string(buf) != "ping" {
		t.Fatalf("tunnel echo: got %q, want ping", buf)
	}
}
