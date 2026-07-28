package proxy

import (
	"bufio"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func StartProxy(hostsFile, listenAddr string) {
	hosts, err := LoadHosts(hostsFile)
	if err != nil {
		Logger.Error("failed to load hosts file", "path", hostsFile, "err", err)
		return
	}
	Logger.Info("hosts loaded", "entries", len(hosts))

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		Logger.Error("failed to bind listener", "addr", listenAddr, "err", err)
		return
	}
	defer listener.Close()
	Logger.Info("proxy listening", "addr", listenAddr)

	done := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		Logger.Info("shutting down")
		close(done)
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-done:
				return
			default:
				Logger.Warn("accept failed", "err", err)
				continue
			}
		}
		go handleClient(conn, hosts)
	}
}

func handleClient(clientConn net.Conn, hosts map[string]string) {
	reader := bufio.NewReader(clientConn)
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		Logger.Warn("failed to read request", "err", err)
		clientConn.Close()
		return
	}
	firstLine = strings.TrimRight(firstLine, "\r\n")

	parts := strings.SplitN(firstLine, " ", 3)
	if len(parts) < 2 {
		Logger.Warn("malformed request line", "line", firstLine)
		clientConn.Close()
		return
	}
	method := parts[0]
	target := parts[1]

	if method == "CONNECT" {
		HandleHTTPSRequest(clientConn, hosts, target)
	} else {
		rewound := RewindReader(reader, firstLine+"\r\n")
		HandleHTTPRequest(&readWriteConn{Conn: clientConn, reader: rewound}, hosts)
	}
}

type readWriteConn struct {
	net.Conn
	reader io.Reader
}

func (r *readWriteConn) Read(p []byte) (int, error) {
	return r.reader.Read(p)
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
