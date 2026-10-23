package proxy

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestSetupLogger(t *testing.T) {
	old := Logger()
	SetupLogger(slog.LevelDebug)
	t.Cleanup(func() { SetLogger(old) })

	if Logger() == nil {
		t.Fatal("Logger is nil after SetupLogger")
	}
	if !Logger().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug level not enabled when configured to debug")
	}
}

func TestSetupLoggerLevelFiltering(t *testing.T) {
	var buf syncBuffer
	old := Logger()
	SetLogger(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { SetLogger(old) })

	Logger().Debug("should not appear")
	Logger().Info("should appear")

	if bytes.Contains([]byte(buf.String()), []byte("should not appear")) {
		t.Error("debug message leaked at info level")
	}
	if !bytes.Contains([]byte(buf.String()), []byte("should appear")) {
		t.Error("info message missing")
	}
}

func TestLoggerConcurrentAccess(t *testing.T) {
	old := Logger()
	t.Cleanup(func() { SetLogger(old) })

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = Logger().Enabled(context.Background(), slog.LevelInfo)
		}()
	}
	SetupLogger(slog.LevelDebug)
	wg.Wait()
}
