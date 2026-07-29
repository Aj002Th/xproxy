# xproxy

基于 hosts 文件的 HTTP/HTTPS 代理服务器。不修改系统 `/etc/hosts` 即可切换域名解析。

## 解决了什么问题

[SwitchHosts](https://github.com/oldj/SwitchHosts) 等工具直接修改系统 hosts 文件，存在两个痛点：

1. **DNS 缓存延迟** — 浏览器和操作系统会缓存 DNS 结果。切换 hosts 后需要等待不确定的时间才能生效，或手动清理缓存。
2. **无法按应用隔离** — 系统 hosts 变更是全局的。无法让浏览器 A 走测试环境、浏览器 B 走正式环境。

xproxy 解决了这两个问题：它作为一个 HTTP/HTTPS 代理服务器运行，根据你的自定义 hosts 文件解析域名。将需要使用的应用配置到该代理上，即可即时切换、按应用隔离。

## 快速开始

```bash
# 编译
make build

# 使用自定义 hosts 文件启动
./xproxy --hosts ./my-hosts --listen 127.0.0.1:8888
```

创建 hosts 文件（`my-hosts`）：

```
# my-hosts
192.168.1.100 api.example.com
10.0.0.50   internal.example.org
```

然后将客户端配置为使用 `127.0.0.1:8888` 作为 HTTP 代理。

## 客户端配置

### Chrome + SwitchyOmega

1. 安装 [SwitchyOmega](https://chrome.google.com/webstore/detail/proxy-switchyomega/padekgcemlokbadohgkifijomclgjgif)
2. 新建代理情景模式：
   - 协议：`HTTP`
   - 服务器：`127.0.0.1`
   - 端口：`8888`
3. 需要使用 hosts 解析时切换到该情景模式即可。

### Postman

Settings → Proxy → 关闭 Use the system proxy，然后配置：
- Type: `HTTP`
- Server: `127.0.0.1`
- Port: `8888`

### curl

```bash
curl -x http://127.0.0.1:8888 http://example.com/api
```

## 命令行参数

```
xproxy [flags]

Flags:
  --hosts string      自定义 hosts 文件路径 (默认 "/etc/hosts")
  --listen string     代理监听地址和端口 (默认 "0.0.0.0:8888")
  --log-level string  日志级别: debug, info, warn, error (默认 "info")
```

## Hosts 文件格式

标准 `/etc/hosts` 格式。每行一条记录：

```
# 注释以 # 开头
127.0.0.1       localhost
192.168.1.1     api.example.com other.example.com
::1             localhost6
2001:db8::1     ipv6.example.com
```

- `#` 开头的行为注释
- 每行格式：`<IP> <域名1> [域名2 ...]`
- 支持 IPv4 和 IPv6
- MVP 阶段暂不支持通配符

## 日志级别

| 级别 | 输出内容 |
|------|----------|
| `debug` | 所有请求详情（非常详细） |
| `info` | 仅 hosts 命中记录和启动信息 |
| `warn` | 上游连接失败 |
| `error` | 启动失败 |

## 工作原理

```
客户端 (Postman)               xproxy                     目标服务器
      │                           │                            │
      │  GET http://api.example   │                            │
      │ ────────────────────────→ │                            │
      │                           │  hosts: api.example.com   │
      │                           │  → 192.168.1.100          │
      │                           │                            │
      │                           │  GET /path HTTP/1.1       │
      │                           │  Host: api.example.com    │
      │                           │ ────────────────────────→ │
      │                           │                            │
      │                           │ ←───────────────────────── │
      │ ←──────────────────────── │                            │
      │  200 OK                   │                            │
```

对于 HTTPS 请求，xproxy 使用 CONNECT 方法建立 TCP 隧道，加密流量透传，不做中间人解密。

## 从源码编译

```bash
git clone https://github.com/Aj002Th/xproxy.git
cd xproxy
make build
```

需要 Go 1.23+。

## Makefile 命令

| 命令 | 说明 |
|------|------|
| `build` | 编译二进制文件 |
| `test` | 运行所有测试 |
| `test-cover` | 测试并生成覆盖率报告 |
| `vet` | 运行 go vet 检查 |
| `clean` | 删除二进制和覆盖率文件 |
| `run` | 编译并以示例配置启动 |
| `install` | 安装到 `$GOPATH/bin` |
| `fmt` | 格式化源代码 |
| `tidy` | 整理 go 模块依赖 |
| `smoke` | 快速冒烟测试 |

## License

MIT
