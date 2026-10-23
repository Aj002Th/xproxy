package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const (
	dialTimeout       = 30 * time.Second
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 90 * time.Second
	maxHeaderBytes    = 64 * 1024
	shutdownTimeout   = 5 * time.Second
)

// StartProxy loads the hosts file and serves the proxy until ctx is canceled.
func StartProxy(ctx context.Context, hostsFile, listenAddr string) error {
	hosts, err := LoadHosts(hostsFile)
	if err != nil {
		return fmt.Errorf("load hosts %q: %w", hostsFile, err)
	}
	Logger().Info("hosts loaded", "entries", len(hosts))

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen %q: %w", listenAddr, err)
	}
	Logger().Info("proxy listening", "addr", listener.Addr().String())

	return serve(ctx, listener, hosts)
}

func serve(ctx context.Context, listener net.Listener, hosts map[string]string) error {
	srv := &http.Server{
		Handler:           NewHandler(hosts),
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(Logger().Handler(), slog.LevelWarn),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		Logger().Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve: %w", err)
	}
}
