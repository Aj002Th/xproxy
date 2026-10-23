package proxy

import (
	"bufio"
	"net"
	"os"
	"strings"
)

// LoadHosts reads an /etc/hosts-style file into a domain-to-IP map.
// Domains are lowercased and trailing dots are removed so lookups are
// case-insensitive, matching DNS semantics.
func LoadHosts(filePath string) (map[string]string, error) {
	hosts := make(map[string]string)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ip := parts[0]
		if net.ParseIP(ip) == nil {
			Logger().Warn("skipping hosts entry with invalid IP", "ip", ip)
			continue
		}
		for _, domain := range parts[1:] {
			hosts[normalizeHost(domain)] = ip
		}
	}
	return hosts, scanner.Err()
}

// normalizeHost lowercases a hostname and strips a single trailing dot.
func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	return strings.TrimSuffix(host, ".")
}
