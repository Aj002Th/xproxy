package proxy

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

// NewHandler returns an HTTP handler that proxies absolute-form requests and
// handles CONNECT tunnels for HTTPS.
func NewHandler(hosts map[string]string) http.Handler {
	reverseProxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// Preserve the client's Host header (the virtual host to reach).
			pr.Out.Host = pr.In.Host
			Logger().Debug("request proxied", "method", pr.In.Method, "host", pr.In.Host)
		},
		Transport: newTransport(hosts),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			Logger().Warn("upstream request failed", "host", r.Host, "err", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		},
		ErrorLog:      slog.NewLogLogger(Logger().Handler(), slog.LevelWarn),
		FlushInterval: -1,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			handleConnect(w, r, hosts)
			return
		}
		if r.URL.Host == "" {
			if r.Host == "" {
				http.Error(w, "missing target host", http.StatusBadRequest)
				return
			}
			r.URL.Host = r.Host
			r.URL.Scheme = "http"
		}
		reverseProxy.ServeHTTP(w, r)
	})
}

func newTransport(hosts map[string]string) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
	}

	return &http.Transport{
		Proxy: nil, // a forward proxy must not chain through the environment
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			lookup := normalizeHost(host)
			if ip, ok := hosts[lookup]; ok {
				Logger().Info("host resolved", "domain", lookup, "ip", ip)
				addr = net.JoinHostPort(ip, port)
			}
			return dialer.DialContext(ctx, network, addr)
		},
		DisableCompression:    true,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
