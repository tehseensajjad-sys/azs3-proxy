package handler

import (
	"io"
	"strings"
	"testing"
)

func TestAwsChunkedReader_ReadsChunks(t *testing.T) {
	payload := "4;chunk-signature=abc\r\nTest\r\n0;chunk-signature=abc\r\n\r\n"
	r := NewAwsChunkedReader(strings.NewReader(payload))

	data, err := io.ReadAll(r)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "Test" {
		t.Fatalf("expected 'Test', got %q", string(data))
	}
}

func TestAwsChunkedReader_MalformedSize(t *testing.T) {
	payload := "zz;chunk-signature=abc\r\ndata\r\n"
	r := NewAwsChunkedReader(strings.NewReader(payload))

	buf := make([]byte, 4)
	if _, err := r.Read(buf); err == nil {
		t.Fatalf("expected error for malformed chunk size")
	}
}
