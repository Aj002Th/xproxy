package proxy

import (
	"io"
	"strings"
	"testing"
)

func TestRewindReader(t *testing.T) {
	r := RewindReader(strings.NewReader("body"), "first line\n")
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(data)
	want := "first line\nbody"
	if got != want {
		t.Errorf("RewindReader: got %q, want %q", got, want)
	}
}

func TestRewindReaderPartialReads(t *testing.T) {
	r := RewindReader(strings.NewReader("world"), "hello ")
	buf := make([]byte, 3)

	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("read1: %v", err)
	}
	if string(buf[:n]) != "hel" {
		t.Errorf("first read: got %q, want hel", buf[:n])
	}

	n, err = r.Read(buf)
	if err != nil {
		t.Fatalf("read2: %v", err)
	}
	if string(buf[:n]) != "lo " {
		t.Errorf("second read: got %q, want lo ", buf[:n])
	}

	n, err = r.Read(buf)
	if err != nil {
		t.Fatalf("read3: %v", err)
	}
	if string(buf[:n]) != "wor" {
		t.Errorf("third read: got %q, want wor", buf[:n])
	}
}
