package azureblob

import (
	"bytes"
	"io"
	"testing"
)

func TestReadSeekCloser(t *testing.T) {
	data := []byte("test data")
	reader := bytes.NewReader(data)
	rsc := &readSeekCloser{Reader: reader}

	// Test Read
	buf := make([]byte, 4)
	n, err := rsc.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 4 {
		t.Errorf("Expected 4 bytes, got %d", n)
	}
	if string(buf) != "test" {
		t.Errorf("Expected 'test', got %q", string(buf))
	}

	// Test Seek
	off, err := rsc.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek failed: %v", err)
	}
	if off != 0 {
		t.Errorf("Expected offset 0, got %d", off)
	}

	// Test Close
	if err := rsc.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
