package proxy

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func handleConnect(w http.ResponseWriter, r *http.Request, hosts map[string]string) {
	target := r.Host
	if target == "" {
		target = r.URL.Host
	}

	host, port, err := parseConnectTarget(target)
	if err != nil {
		Logger().Warn("invalid CONNECT target", "target", target, "err", err)
		http.Error(w, "invalid CONNECT target", http.StatusBadRequest)
		return
	}

	originalHost := host
	dialHost := normalizeHost(host)
	if ip, ok := hosts[dialHost]; ok {
		dialHost = ip
		Logger().Info("host resolved", "domain", originalHost, "ip", ip)
	}

	upstream, err := net.DialTimeout("tcp", net.JoinHostPort(dialHost, port), dialTimeout)
	if err != nil {
		Logger().Warn("upstream unreachable", "target", net.JoinHostPort(dialHost, port), "err", err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	defer upstream.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, clientRW, err := hijacker.Hijack()
	if err != nil {
		Logger().Warn("failed to hijack client connection", "err", err)
		return
	}
	defer clientConn.Close()

	// The server may have set a header-read deadline; the tunnel is long-lived.
	_ = clientConn.SetDeadline(time.Time{})

	if _, err := io.WriteString(clientRW, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		Logger().Warn("failed to send CONNECT response", "err", err)
		return
	}
	if err := clientRW.Flush(); err != nil {
		Logger().Warn("failed to flush CONNECT response", "err", err)
		return
	}

	Logger().Debug("tunnel established", "host", originalHost, "port", port)
	tunnel(clientConn, upstream, clientRW.Reader, upstream)
}

// parseConnectTarget accepts host:port, host (defaults to 443), and bracketed
// IPv6 literals.
func parseConnectTarget(target string) (host, port string, err error) {
	if target == "" {
		return "", "", errors.New("empty CONNECT target")
	}

	host, port, err = net.SplitHostPort(target)
	if err == nil {
		if host == "" {
			return "", "", errors.New("empty host")
		}
		return host, DefaultPort(port, "443"), nil
	}

	var addrErr *net.AddrError
	if errors.As(err, &addrErr) && strings.Contains(addrErr.Err, "missing port") {
		host = strings.Trim(target, "[]")
		if host == "" {
			return "", "", errors.New("empty host")
		}
		return host, "443", nil
	}
	return "", "", err
}
