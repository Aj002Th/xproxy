package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/Aj002Th/xproxy/internal/proxy"
	"github.com/spf13/cobra"
)

var (
	hostsFile  string
	listenAddr string
)

var rootCmd = &cobra.Command{
	Use:   "xproxy",
	Short: "A cross-platform transparent proxy server",
	Long:  `A transparent proxy server that forwards HTTP/HTTPS traffic based on rules in a hosts file.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 检查平台兼容性
		if runtime.GOOS == "windows" {
			log.Println("Warning: Transparent proxy is not fully supported on Windows.")
		}

		// 启动代理服务器
		proxy.StartProxy(hostsFile, listenAddr)
	},
}

func init() {
	rootCmd.Flags().StringVar(&hostsFile, "hosts", "/etc/hosts", "Path to the hosts file")
	rootCmd.Flags().StringVar(&listenAddr, "listen", "0.0.0.0:8888", "Address and port to listen on")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
