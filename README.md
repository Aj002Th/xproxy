# xproxy

A hosts-based HTTP/HTTPS proxy server. Switch domains without touching `/etc/hosts`.

## Why xproxy

Tools like [SwitchHosts](https://github.com/oldj/SwitchHosts) modify the system hosts file directly, which causes two problems:

1. **DNS cache lag** — browsers and the OS cache DNS results. After switching hosts, you wait an unpredictable amount of time for changes to take effect (or manually flush caches).
2. **No per-app isolation** — system hosts changes are global. You can't have one browser use a test environment while another uses production.

xproxy solves both: it exposes an HTTP/HTTPS proxy server that resolves domains using your custom hosts file. Route individual applications through the proxy and switch hosts entries instantly, per app.

## Quick Start

```bash
# Build
make build

# Start with a custom hosts file
./xproxy --hosts ./my-hosts --listen 127.0.0.1:8888
```

Create a hosts file (`my-hosts`):

```
# my-hosts
192.168.1.100 api.example.com
10.0.0.50   internal.example.org
```

Then configure your client to use `127.0.0.1:8888` as an HTTP proxy.

## Client Configuration

### Chrome + SwitchyOmega

1. Install [SwitchyOmega](https://chrome.google.com/webstore/detail/proxy-switchyomega/padekgcemlokbadohgkifijomclgjgif)
2. Create a new proxy profile:
   - Protocol: `HTTP`
   - Server: `127.0.0.1`
   - Port: `8888`
3. Switch to that profile when you need hosts-based resolution.

### Postman

Settings → Proxy → Use the system proxy → Off. Then configure:
- Type: `HTTP`
- Server: `127.0.0.1`
- Port: `8888`

### curl

```bash
curl -x http://127.0.0.1:8888 http://example.com/api
```

## CLI Usage

```
xproxy [flags]

Flags:
  --hosts string      path to hosts file (default "/etc/hosts")
  --listen string     address and port to listen on (default "0.0.0.0:8888")
  --log-level string  log level: debug, info, warn, error (default "info")
```

## Hosts File Format

Standard `/etc/hosts` format. One entry per line:

```
# comments start with #
127.0.0.1       localhost
192.168.1.1     api.example.com other.example.com
::1             localhost6
2001:db8::1     ipv6.example.com
```

- Comments start with `#`
- Each line: `<IP> <domain1> [domain2 ...]`
- IPv4 and IPv6 supported
- No wildcards (MVP)

## Logging

| Level | What you see |
|-------|-------------|
| `debug` | Every request proxied (very verbose) |
| `info` | Only hosts hits and startup events |
| `warn` | Upstream connection failures |
| `error` | Startup failures |

## How It Works

```
Client (Postman)                  xproxy                     Target Server
     │                              │                              │
     │  GET http://api.example.com  │                              │
     │ ──────────────────────────→ │                              │
     │                              │  hosts: api.example.com     │
     │                              │  → 192.168.1.100           │
     │                              │                              │
     │                              │  GET /path HTTP/1.1         │
     │                              │  Host: api.example.com      │
     │                              │ ──────────────────────────→ │
     │                              │                              │
     │                              │ ←────────────────────────── │
     │ ←────────────────────────── │                              │
     │  200 OK                     │                              │
```

For HTTPS, xproxy uses the CONNECT method to establish a TCP tunnel, then relays encrypted traffic transparently.

## Build from Source

```bash
git clone https://github.com/Aj002Th/xproxy.git
cd xproxy
make build
```

Requires Go 1.23+.

## Makefile Targets

| Target | Description |
|--------|-------------|
| `build` | Compile binary |
| `test` | Run all tests |
| `test-cover` | Tests with coverage report |
| `vet` | Run go vet |
| `clean` | Remove binary and coverage output |
| `run` | Build and start with example hosts |
| `install` | Install to `$GOPATH/bin` |
| `fmt` | Format source code |
| `tidy` | Tidy go modules |
| `smoke` | Quick smoke test |

## License

MIT
