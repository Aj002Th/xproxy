package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"
)

// 解析 hosts 文件映射
var hostsMap = map[string]string{
	"example.com": "1.2.3.4",
}

// 解析 hosts 文件
func loadHosts(filePath string) map[string]string {
	hosts := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open hosts file: %v", err)
		return hosts
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 忽略空行和注释（# 开头的行）
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 按空格或制表符分割
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		ip := fields[0]
		for _, domain := range fields[1:] {
			hosts[domain] = ip
			log.Printf("Loaded host: %s -> %s", domain, ip)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading hosts file: %v", err)
	}

	return hosts
}

// 记录日志
func logRequest(protocol, method, originalHost, targetHost string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	log.Printf("[%s] [%s] %s %s -> %s\n", timestamp, protocol, method, originalHost, targetHost)
}

// 修改 HTTP 请求的目标地址
func modifyRequest(req *http.Request) {
	originalHost := req.URL.Host
	if ip, exists := hostsMap[req.URL.Hostname()]; exists {
		targetHost := fmt.Sprintf("%s:%s", ip, req.URL.Port())
		req.URL.Host = targetHost
		req.Host = req.URL.Hostname()
		logRequest("HTTP", req.Method, originalHost, targetHost)
	}
}

// 处理 HTTP 代理
func handleHTTPProxy(w http.ResponseWriter, r *http.Request) {
	proxy := &httputil.ReverseProxy{
		Director: modifyRequest,
	}
	proxy.ServeHTTP(w, r)
}

// 处理 HTTPS 代理
func handleHTTPSProxy(w http.ResponseWriter, r *http.Request) {
	originalHost := r.Host
	targetHost := r.Host
	if ip, exists := hostsMap[strings.Split(r.Host, ":")[0]]; exists {
		targetHost = fmt.Sprintf("%s:%s", ip, strings.Split(r.Host, ":")[1])
	}

	logRequest("HTTPS", "CONNECT", originalHost, targetHost)

	targetConn, err := net.Dial("tcp", targetHost)
	if err != nil {
		http.Error(w, "Failed to connect to target", http.StatusServiceUnavailable)
		return
	}
	defer targetConn.Close()

	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "Hijacking failed", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	go io.Copy(targetConn, clientConn)
	go io.Copy(clientConn, targetConn)
}

func main() {
	// 解析 hosts 文件
	hostsMap = loadHosts("./hosts")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			handleHTTPSProxy(w, r)
		} else {
			handleHTTPProxy(w, r)
		}
	})

	log.Println("Proxy server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
