package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/Aj002Th/xproxy/internal/service"
	"github.com/spf13/cobra"
)

var (
	hostsFile  string
	listenAddr string
	logLevel   string
	version    string
)

var rootCmd = &cobra.Command{
	Use:     "xproxy",
	Short:   "A hosts-based HTTP/HTTPS proxy server",
	Long:    `xproxy reads a hosts file and exposes an HTTP/HTTPS proxy. Configure your browser or Postman to use this proxy, and domains in the hosts file will be resolved to their mapped IPs without modifying the system hosts file.`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		level := parseLogLevel(logLevel)
		proxy.SetupLogger(level)
		proxy.Logger().Info("starting xproxy", "hosts", hostsFile, "listen", listenAddr)
		proxy.StartProxy(hostsFile, listenAddr)
	},
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		fmt.Fprintf(os.Stderr, "unknown log level %q, using info\n", s)
		return slog.LevelInfo
	}
}

func init() {
	rootCmd.Flags().StringVar(&hostsFile, "hosts", "/etc/hosts", "path to hosts file")
	rootCmd.Flags().StringVar(&listenAddr, "listen", "127.0.0.1:8888", "address and port to listen on")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "log level: debug, info, warn, error")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
