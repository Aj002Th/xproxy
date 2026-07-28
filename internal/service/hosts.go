package proxy

import (
	"bufio"
	"os"
	"strings"
)

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
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ip := parts[0]
		for _, domain := range parts[1:] {
			hosts[domain] = ip
		}
	}
	return hosts, scanner.Err()
}
