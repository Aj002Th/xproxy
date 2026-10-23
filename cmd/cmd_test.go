package cmd

import (
	"log/slog"
	"testing"
)

func TestFlagDefaults(t *testing.T) {
	if got := rootCmd.Flags().Lookup("listen").DefValue; got != "127.0.0.1:8888" {
		t.Errorf("listen default = %q, want 127.0.0.1:8888", got)
	}
	if got := rootCmd.Flags().Lookup("hosts").DefValue; got != "/etc/hosts" {
		t.Errorf("hosts default = %q, want /etc/hosts", got)
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}
	for _, tc := range tests {
		got := parseLogLevel(tc.input)
		if got != tc.want {
			t.Errorf("parseLogLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
