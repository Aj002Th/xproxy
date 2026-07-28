package proxy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts")
	content := `# comment
127.0.0.1 localhost
192.168.1.1 api.example.com other.example.com

10.0.0.1 single.test
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hosts, err := LoadHosts(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hosts["localhost"] != "127.0.0.1" {
		t.Errorf("localhost: got %q, want 127.0.0.1", hosts["localhost"])
	}
	if hosts["api.example.com"] != "192.168.1.1" {
		t.Errorf("api.example.com: got %q, want 192.168.1.1", hosts["api.example.com"])
	}
	if hosts["other.example.com"] != "192.168.1.1" {
		t.Errorf("other.example.com: got %q, want 192.168.1.1", hosts["other.example.com"])
	}
	if hosts["single.test"] != "10.0.0.1" {
		t.Errorf("single.test: got %q, want 10.0.0.1", hosts["single.test"])
	}
	if _, exists := hosts["example.com"]; exists {
		t.Error("comment line domain should not exist")
	}
	if _, exists := hosts["#"]; exists {
		t.Error("comment character should not appear as domain")
	}
}

func TestLoadHostsFileNotExist(t *testing.T) {
	_, err := LoadHosts("/nonexistent/hosts/file")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadHostsIPv6(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts")
	content := "::1 localhost6\n2001:db8::1 ipv6.example.com\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hosts, err := LoadHosts(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hosts["localhost6"] != "::1" {
		t.Errorf("localhost6: got %q, want ::1", hosts["localhost6"])
	}
	if hosts["ipv6.example.com"] != "2001:db8::1" {
		t.Errorf("ipv6.example.com: got %q, want 2001:db8::1", hosts["ipv6.example.com"])
	}
}
