package azureblob

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
)

func TestNewConcurrentReader_ZeroSize(t *testing.T) {
	ctx := context.Background()

	r := newConcurrentReader(ctx, nil, 0, 0, 0)
	defer func() { _ = r.Close() }()

	buf := make([]byte, 8)
	n, err := r.Read(buf)
	if n != 0 {
		t.Fatalf("expected zero bytes read, got %d", n)
	}
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestConcurrentReader_ReadsData(t *testing.T) {
	data := []byte("hello world")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			rangeHeader = r.Header.Get("x-ms-range")
		}
		start := 0
		end := len(data) - 1
		if rangeHeader != "" {
			_, _ = fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
			if end >= len(data) || end < 0 {
				end = len(data) - 1
			}
		}

		w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(data[start : end+1])
	}))
	defer srv.Close()

	client, err := blockblob.NewClientWithNoCredential(srv.URL+"/container/blob", &blockblob.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: srv.Client(),
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	r := newConcurrentReader(context.Background(), client, int64(len(data)), 1, 4)
	defer func() { _ = r.Close() }()

	readData, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected error reading: %v", err)
	}
	if string(readData) != string(data) {
		t.Fatalf("expected %q, got %q", string(data), string(readData))
	}
}
