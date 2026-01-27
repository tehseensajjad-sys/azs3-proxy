package common

import (
	"bytes"
	"io"
	"testing"
)

func TestReadSeekCloser(t *testing.T) {
	data := []byte("hello")
	rsc := &ReadSeekCloser{Reader: bytes.NewReader(data)}

	buf := make([]byte, len(data))
	n, err := rsc.Read(buf)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if n != len(data) || string(buf[:n]) != "hello" {
		t.Fatalf("unexpected read data: %q", string(buf[:n]))
	}

	if _, err := rsc.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek failed: %v", err)
	}

	if err := rsc.Close(); err != nil {
		t.Fatalf("close should be no-op, got %v", err)
	}
}
