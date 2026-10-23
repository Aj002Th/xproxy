package proxy

import (
	"io"
	"net"
)

func DefaultPort(port, defaultPort string) string {
	if port == "" {
		return defaultPort
	}
	return port
}

// tunnel relays bytes in both directions and closes both connections once
// either direction finishes. clientReader may be a buffered reader that still
// holds bytes read past the request headers.
func tunnel(client, server net.Conn, clientReader, serverReader io.Reader) {
	go func() {
		defer server.Close()
		defer client.Close()
		io.Copy(server, clientReader)
	}()
	io.Copy(client, serverReader)
}
