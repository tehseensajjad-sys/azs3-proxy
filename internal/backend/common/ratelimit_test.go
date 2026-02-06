package common

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestBandwidthLimiterWrapsDownloadUpload(t *testing.T) {
	data := []byte("hello world")

	limiter := NewBandwidthLimiter(10, 10, 10)

	// Download wrapper
	dl := limiter.WrapDownload(io.NopCloser(bytes.NewReader(data)))
	out, err := io.ReadAll(dl)
	if err != nil {
		t.Fatalf("download read failed: %v", err)
	}
	if string(out) != string(data) {
		t.Fatalf("download data mismatch: %q", string(out))
	}
	_ = dl.Close()

	// Upload wrapper
	up := limiter.WrapUpload(bytes.NewReader(data))
	out, err = io.ReadAll(up)
	if err != nil {
		t.Fatalf("upload read failed: %v", err)
	}
	if string(out) != string(data) {
		t.Fatalf("upload data mismatch: %q", string(out))
	}

	// Upload seekable wrapper
	upSeek := limiter.WrapUploadSeek(&ReadSeekCloser{Reader: bytes.NewReader(data)})
	out, err = io.ReadAll(upSeek)
	if err != nil {
		t.Fatalf("upload seek read failed: %v", err)
	}
	if string(out) != string(data) {
		t.Fatalf("upload seek data mismatch: %q", string(out))
	}
	if _, err := upSeek.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek failed: %v", err)
	}
}

func TestBandwidthLimiterNil(t *testing.T) {
	var limiter *BandwidthLimiter
	data := []byte("x")

	dl := limiter.WrapDownload(io.NopCloser(bytes.NewReader(data)))
	out, err := io.ReadAll(dl)
	if err != nil {
		t.Fatalf("nil download read failed: %v", err)
	}
	if string(out) != "x" {
		t.Fatalf("nil download mismatch: %q", string(out))
	}
	_ = dl.Close()

	if limiter.WrapUpload(nil) != nil {
		t.Fatalf("nil upload should return nil when input nil")
	}

	if got := NewBandwidthLimiter(0, 0, 0); got != nil {
		t.Fatalf("expected nil limiter for zero caps")
	}
}

func TestBandwidthLimiterEnforcesRate(t *testing.T) {
	// 1 Mbps ~ 125kB/s; 256kB should take a bit over 1s given initial tokens.
	buf := bytes.Repeat([]byte{'a'}, 256*1024)
	limiter := NewBandwidthLimiter(1, 1, 1)

	start := time.Now()
	if _, err := io.Copy(io.Discard, limiter.WrapDownload(io.NopCloser(bytes.NewReader(buf)))); err != nil {
		t.Fatalf("download copy failed: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 400*time.Millisecond {
		t.Fatalf("expected download to be throttled, finished in %v", elapsed)
	}

	start = time.Now()
	if _, err := io.Copy(io.Discard, limiter.WrapUpload(bytes.NewReader(buf))); err != nil {
		t.Fatalf("upload copy failed: %v", err)
	}
	elapsed = time.Since(start)
	if elapsed < 400*time.Millisecond {
		t.Fatalf("expected upload to be throttled, finished in %v", elapsed)
	}
}
