package proxy

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestSetupLogger(t *testing.T) {
	SetupLogger(slog.LevelDebug)
	if Logger == nil {
		t.Fatal("Logger is nil after SetupLogger")
	}
	if !Logger.Enabled(nil, slog.LevelDebug) {
		t.Error("debug level not enabled when configured to debug")
	}
}

func TestSetupLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	Logger.Debug("should not appear")
	Logger.Info("should appear")
	if bytes.Contains(buf.Bytes(), []byte("should not appear")) {
		t.Error("debug message leaked at info level")
	}
	if !bytes.Contains(buf.Bytes(), []byte("should appear")) {
		t.Error("info message missing at info level")
	}
}
