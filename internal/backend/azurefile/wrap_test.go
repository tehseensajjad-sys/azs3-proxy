package azurefile

import (
	"bytes"
	"io"
	"testing"

	backendcommon "github.com/vibhansa-msft/azs3-proxy/internal/backend/common"
)

func TestWrapDownloadUploadWithLimiter(t *testing.T) {
	limiter := backendcommon.NewBandwidthLimiter(10, 10, 0)
	af := &AzureFileBackend{limiter: limiter}
	data := []byte("hello")

	rc := io.NopCloser(bytes.NewReader(data))
	wrapped := af.wrapDownload(rc)
	out, err := io.ReadAll(wrapped)
	if err != nil {
		t.Fatalf("wrapDownload read failed: %v", err)
	}
	if string(out) != string(data) {
		t.Fatalf("download mismatch: %s", string(out))
	}
	_ = wrapped.Close()

	r := af.wrapUpload(bytes.NewReader(data))
	out, err = io.ReadAll(r)
	if err != nil {
		t.Fatalf("wrapUpload read failed: %v", err)
	}
	if string(out) != string(data) {
		t.Fatalf("upload mismatch: %s", string(out))
	}
}

func TestWrapDownloadUploadNoLimiter(t *testing.T) {
	af := &AzureFileBackend{}
	data := []byte("abc")

	rc := io.NopCloser(bytes.NewReader(data))
	if got := af.wrapDownload(rc); got != rc {
		t.Fatalf("wrapDownload should return original when limiter nil")
	}

	r := bytes.NewReader(data)
	if got := af.wrapUpload(r); got != r {
		t.Fatalf("wrapUpload should return original when limiter nil")
	}
}
