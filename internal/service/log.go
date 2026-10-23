package proxy

import (
	"log/slog"
	"os"
	"sync/atomic"
)

var logger atomic.Pointer[slog.Logger]

func init() {
	logger.Store(slog.Default())
}

// Logger returns the process-wide logger. It is safe for concurrent use.
func Logger() *slog.Logger {
	return logger.Load()
}

// SetupLogger configures the process-wide logger with the given level.
func SetupLogger(level slog.Level) {
	logger.Store(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}

// SetLogger replaces the process-wide logger. Intended for tests and embedding.
func SetLogger(l *slog.Logger) {
	logger.Store(l)
}
