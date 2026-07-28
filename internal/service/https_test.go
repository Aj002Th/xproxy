package proxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestRewindReader(t *testing.T) {
	r := RewindReader(strings.NewReader("body"), "first line\n")
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(data)
	want := "first line\nbody"
	if got != want {
		t.Errorf("RewindReader: got %q, want %q", got, want)
	}
}

func TestRewindReaderPartialReads(t *testing.T) {
	r := RewindReader(strings.NewReader("world"), "hello ")
	buf := make([]byte, 3)

	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("read1: %v", err)
	}
	if string(buf[:n]) != "hel" {
		t.Errorf("first read: got %q, want hel", buf[:n])
	}

	n, err = r.Read(buf)
	if err != nil {
		t.Fatalf("read2: %v", err)
	}
	if string(buf[:n]) != "lo " {
		t.Errorf("second read: got %q, want lo ", buf[:n])
	}

	n, err = r.Read(buf)
	if err != nil {
		t.Fatalf("read3: %v", err)
	}
	if string(buf[:n]) != "wor" {
		t.Errorf("third read: got %q, want wor", buf[:n])
	}
}

func TestHandleHTTPSRequestTunnel(t *testing.T) {
	upstream, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start upstream: %v", err)
	}
	defer upstream.Close()

	upstreamHost := upstream.Addr().String()

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

	hosts := map[string]string{
		"example.com": "127.0.0.1",
	}

	go HandleHTTPSRequest(serverConn, hosts, upstreamHost)

	resp, err := http.ReadResponse(bufio.NewReader(clientConn), nil)
	if err != nil {
		t.Fatalf("failed to read CONNECT response: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 Connection Established, got %d", resp.StatusCode)
	}

	msg := []byte("hello through tunnel")
	clientConn.Write(msg)

	buf := make([]byte, len(msg))
	_, err = io.ReadFull(clientConn, buf)
	if err != nil {
		t.Fatalf("failed to read echoed data: %v", err)
	}
	if string(buf) != string(msg) {
		t.Errorf("echo: got %q, want %q", string(buf), string(msg))
	}
	clientConn.Close()
}

func TestHandleHTTPSRequestWithoutPort(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go HandleHTTPSRequest(serverConn, map[string]string{}, "127.0.0.1:19998")

	resp, err := http.ReadResponse(bufio.NewReader(clientConn), nil)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 502 {
		t.Errorf("expected 502, got %d", resp.StatusCode)
	}
}

func TestHandleHTTPSRequestPortDefaulted(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go HandleHTTPSRequest(serverConn, map[string]string{"example.com": "127.0.0.1"}, "example.com")

	resp, err := http.ReadResponse(bufio.NewReader(clientConn), nil)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 502 {
		t.Errorf("expected 502 for port-defaulted connect to closed port, got %d", resp.StatusCode)
	}
}
