package proxy

import "testing"

func TestDefaultPort(t *testing.T) {
	tests := []struct {
		port, def, want string
	}{
		{"", "80", "80"},
		{"8080", "80", "8080"},
		{"443", "80", "443"},
	}
	for _, tc := range tests {
		if got := DefaultPort(tc.port, tc.def); got != tc.want {
			t.Errorf("DefaultPort(%q, %q) = %q, want %q", tc.port, tc.def, got, tc.want)
		}
	}
}
