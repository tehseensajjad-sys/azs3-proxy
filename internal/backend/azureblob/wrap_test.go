package azureblob

import (
	"bytes"
	"io"
	"testing"

	backendcommon "github.com/vibhansa-msft/azs3-proxy/internal/backend/common"
)

func TestWrapDownloadUploadWithLimiter(t *testing.T) {
	limiter := backendcommon.NewBandwidthLimiter(10, 10, 0)
	b := &AzureBlobBackend{limiter: limiter}

	data := []byte("hello world")

	// download wrapper
	rc := io.NopCloser(bytes.NewReader(data))
	wrapped := b.wrapDownload(rc)
	got, err := io.ReadAll(wrapped)
	if err != nil {
		t.Fatalf("wrapDownload read failed: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("download mismatch: %s", string(got))
	}
	_ = wrapped.Close()

	// upload wrapper
	up := b.wrapUpload(bytes.NewReader(data))
	got, err = io.ReadAll(up)
	if err != nil {
		t.Fatalf("wrapUpload read failed: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("upload mismatch: %s", string(got))
	}

	// upload seek wrapper
	seeker := &backendcommon.ReadSeekCloser{Reader: bytes.NewReader(data)}
	upSeek := b.wrapUploadSeek(seeker)
	got, err = io.ReadAll(upSeek)
	if err != nil {
		t.Fatalf("wrapUploadSeek read failed: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("upload seek mismatch: %s", string(got))
	}
	if _, err := upSeek.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("wrapUploadSeek seek failed: %v", err)
	}
}

func TestWrapDownloadUploadNoLimiter(t *testing.T) {
	b := &AzureBlobBackend{}
	data := []byte("abc")

	rc := io.NopCloser(bytes.NewReader(data))
	if got := b.wrapDownload(rc); got != rc {
		t.Fatalf("wrapDownload should return original when limiter nil")
	}

	r := bytes.NewReader(data)
	if got := b.wrapUpload(r); got != r {
		t.Fatalf("wrapUpload should return original when limiter nil")
	}

	seeker := &backendcommon.ReadSeekCloser{Reader: bytes.NewReader(data)}
	if got := b.wrapUploadSeek(seeker); got != seeker {
		t.Fatalf("wrapUploadSeek should return original when limiter nil")
	}
}
