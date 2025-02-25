package proxy

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

// StartProxy 启动代理服务器
func StartProxy(hostsFile, listenAddr string) {
	// 加载 hosts 文件
	hosts, err := loadHosts(hostsFile)
	if err != nil {
		log.Fatalf("Failed to load hosts file: %v", err)
	}
	fmt.Printf("Loaded hosts: %v\n", hosts)

	// 启动代理服务器
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to start proxy server: %v", err)
	}
	defer listener.Close()
	fmt.Printf("Proxy server started on %s\n", listenAddr)

	// 接受客户端连接
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept client connection: %v", err)
			continue
		}
		go handleClient(clientConn.(*net.TCPConn), hosts)
	}
}

// 加载 hosts 文件
func loadHosts(filePath string) (map[string]string, error) {
	hosts := make(map[string]string)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // 忽略空行和注释
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			ip := parts[0]
			for _, domain := range parts[1:] {
				hosts[domain] = ip
			}
		}
	}
	return hosts, nil
}

// 处理客户端连接
func handleClient(clientConn *net.TCPConn, hosts map[string]string) {
	// 读取客户端请求的第一个字节以判断协议
	buf := make([]byte, 1)
	_, err := clientConn.Read(buf)
	if err != nil {
		log.Printf("Failed to read client request: %v", err)
		return
	}

	// 回退读取的字节
	clientConn = &rewindConn{clientConn, buf}

	// 判断是 HTTP 还是 HTTPS 请求
	if buf[0] == 'C' { // CONNECT 方法的第一个字母是 'C'
		handleHTTPSRequest(clientConn, hosts)
	} else {
		handleHTTPRequest(clientConn, hosts)
	}
}

// rewindConn 用于回退读取的字节
type rewindConn struct {
	net.Conn
	buf []byte
}

func (r *rewindConn) Read(p []byte) (n int, err error) {
	if len(r.buf) > 0 {
		n = copy(p, r.buf)
		r.buf = r.buf[n:]
		return n, nil
	}
	return r.Conn.Read(p)
}

// 处理 HTTP 请求
func handleHTTPRequest(clientConn net.Conn, hosts map[string]string) {
	defer clientConn.Close()

	// 解析 HTTP 请求
	request, err := http.ReadRequest(bufio.NewReader(clientConn))
	if err != nil {
		log.Printf("Failed to parse HTTP request: %v", err)
		return
	}

	// 查找 hosts 文件中的映射
	host := request.URL.Hostname()
	if ip, ok := hosts[host]; ok {
		request.URL.Host = ip + ":" + request.URL.Port()
		fmt.Printf("Resolved %s to %s\n", host, ip)
	}

	// 创建与目标服务器的连接
	serverConn, err := net.Dial("tcp", request.URL.Host)
	if err != nil {
		log.Printf("Failed to connect to server: %v", err)
		return
	}
	defer serverConn.Close()

	// 发送请求到目标服务器
	if err := request.Write(serverConn); err != nil {
		log.Printf("Failed to send request to server: %v", err)
		return
	}

	// 将目标服务器的响应转发给客户端
	io.Copy(clientConn, serverConn)
}

// 处理 HTTPS 请求
func handleHTTPSRequest(clientConn *net.TCPConn, hosts map[string]string) {
	defer clientConn.Close()

	// 获取原始目标地址
	target, err := getOriginalDst(clientConn)
	if err != nil {
		log.Printf("Failed to get original destination: %v", err)
		return
	}

	// 查找 hosts 文件中的映射
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		log.Printf("Invalid target address: %v", err)
		return
	}
	if ip, ok := hosts[host]; ok {
		host = ip
		fmt.Printf("Resolved %s to %s\n", host, ip)
	}

	// 创建与目标服务器的连接
	serverConn, err := net.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		log.Printf("Failed to connect to server: %v", err)
		return
	}
	defer serverConn.Close()

	// 返回 200 响应，表示隧道建立成功
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// 双向转发数据
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(clientConn, serverConn)
	}()
	go func() {
		defer wg.Done()
		io.Copy(serverConn, clientConn)
	}()
	wg.Wait()
}
