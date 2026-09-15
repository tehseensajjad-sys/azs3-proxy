package azurefile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"
	"github.com/vibhansa-msft/azs3-proxy/internal/backend"
	"go.uber.org/zap"
)

// TestAzureFileBackendImplementsInterface verifies that AzureFileBackend
// implements the StorageBackend interface at compile time.
func TestAzureFileBackendImplementsInterface(t *testing.T) {
	var _ backend.StorageBackend = (*AzureFileBackend)(nil)
	t.Log("AzureFileBackend successfully implements StorageBackend interface")
}

func TestUploadPartValidations(t *testing.T) {
	backend := &AzureFileBackend{multipartUploads: make(map[string]*MultipartUploadMetadata)}

	// Oversized part should fail early.
	if _, err := backend.UploadPart(context.Background(), "b", "k", "id", 1, maxPartSize+1, strings.NewReader("")); err == nil {
		t.Fatalf("expected error for oversized part")
	}

	// Unknown upload ID should error.
	if _, err := backend.UploadPart(context.Background(), "b", "k", "missing", 1, 1, strings.NewReader("a")); err == nil {
		t.Fatalf("expected error for missing upload")
	}

	// Mismatched bucket/key should error.
	upload := &MultipartUploadMetadata{ShareName: "bucket", FilePath: "key", Parts: map[int][]byte{}, PartOrder: []int{}}
	backend.multipartUploads["upload1"] = upload
	if _, err := backend.UploadPart(context.Background(), "other", "k", "upload1", 1, 1, strings.NewReader("a")); err == nil {
		t.Fatalf("expected error for bucket/key mismatch")
	}

	// Smaller uploads succeed but we avoid multi-GB allocations in tests.
	upload.Parts[1] = bytes.Repeat([]byte{'x'}, 10)
}

func TestUploadPartStoresDataAndOrder(t *testing.T) {
	backend := &AzureFileBackend{multipartUploads: make(map[string]*MultipartUploadMetadata)}
	backend.multipartUploads["upload1"] = &MultipartUploadMetadata{
		ShareName: "bucket",
		FilePath:  "key",
		Parts:     map[int][]byte{},
		PartOrder: []int{},
	}

	etag, err := backend.UploadPart(context.Background(), "bucket", "key", "upload1", 1, 3, strings.NewReader("abc"))
	if err != nil || etag == "" {
		t.Fatalf("expected successful upload part, got err=%v etag=%s", err, etag)
	}

	// Re-upload same part number should overwrite data without duplicating order.
	if _, err := backend.UploadPart(context.Background(), "bucket", "key", "upload1", 1, 3, strings.NewReader("xyz")); err != nil {
		t.Fatalf("unexpected error overwriting part: %v", err)
	}
	meta := backend.multipartUploads["upload1"]
	if got := string(meta.Parts[1]); got != "xyz" {
		t.Fatalf("expected data overwritten to xyz, got %s", got)
	}
	if len(meta.PartOrder) != 1 || meta.PartOrder[0] != 1 {
		t.Fatalf("expected single part order [1], got %v", meta.PartOrder)
	}

	// Add another part and ensure ordering preserved.
	if _, err := backend.UploadPart(context.Background(), "bucket", "key", "upload1", 2, 2, strings.NewReader("qq")); err != nil {
		t.Fatalf("unexpected error uploading second part: %v", err)
	}
	if len(meta.PartOrder) != 2 || meta.PartOrder[0] != 1 || meta.PartOrder[1] != 2 {
		t.Fatalf("expected part order [1 2], got %v", meta.PartOrder)
	}
}

func TestMultipartMetadataOperations(t *testing.T) {
	backend := &AzureFileBackend{multipartUploads: make(map[string]*MultipartUploadMetadata)}
	backend.multipartUploads["u1"] = &MultipartUploadMetadata{ShareName: "bucket", FilePath: "key", Parts: map[int][]byte{1: []byte("a")}, PartOrder: []int{1}}
	backend.multipartUploads["u2"] = &MultipartUploadMetadata{ShareName: "other", FilePath: "key2", Parts: map[int][]byte{2: []byte("bb")}, PartOrder: []int{2}}

	parts, err := backend.ListParts(context.Background(), "bucket", "key", "u1")
	if err != nil || len(parts) != 1 {
		t.Fatalf("expected one part, got %d err=%v", len(parts), err)
	}

	uploads, err := backend.ListMultipartUploads(context.Background(), "bucket")
	if err != nil {
		t.Fatalf("unexpected error listing uploads: %v", err)
	}
	if len(uploads) != 1 {
		t.Fatalf("expected one upload for bucket, got %d", len(uploads))
	}

	if err := backend.AbortMultipartUpload(context.Background(), "bucket", "key", "u1"); err != nil {
		t.Fatalf("expected successful abort: %v", err)
	}
	if _, ok := backend.multipartUploads["u1"]; ok {
		t.Fatalf("expected upload u1 removed after abort")
	}

	if err := backend.AbortMultipartUpload(context.Background(), "bucket", "key", "missing"); err == nil {
		t.Fatalf("expected error aborting missing upload")
	}
}

func TestInitiateMultipartUploadStoresMetadata(t *testing.T) {
	client, err := service.NewClientWithNoCredential("https://example.com", nil)
	if err != nil {
		t.Fatalf("failed to build dummy service client: %v", err)
	}
	backend := &AzureFileBackend{
		client:           client,
		shareClients:     sync.Map{},
		multipartUploads: make(map[string]*MultipartUploadMetadata),
	}

	uploadID, err := backend.InitiateMultipartUpload(context.Background(), "bucket", "file.txt")
	if err != nil {
		t.Fatalf("unexpected error initiating multipart upload: %v", err)
	}
	meta, ok := backend.multipartUploads[uploadID]
	if !ok {
		t.Fatalf("metadata not stored for uploadID %s", uploadID)
	}
	if meta.ShareName != "bucket" || meta.FilePath != "file.txt" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}
}

func TestCreateParentDirectoriesNoopForRoot(t *testing.T) {
	client, err := service.NewClientWithNoCredential("https://example.com", nil)
	if err != nil {
		t.Fatalf("failed to build dummy service client: %v", err)
	}
	shareClient := client.NewShareClient("share")
	backend := &AzureFileBackend{}

	if err := backend.createParentDirectories(context.Background(), shareClient, "object.txt"); err != nil {
		t.Fatalf("expected no-op for root path, got %v", err)
	}
}

// errorTransport prevents outbound network calls by returning an empty response.
type errorTransport struct{}

func (errorTransport) Do(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}
	return resp, nil
}

func TestAzureFileOperationsWithFakeTransport(t *testing.T) {
	client, err := service.NewClientWithNoCredential("https://example.com", &service.ClientOptions{ClientOptions: azcore.ClientOptions{Transport: errorTransport{}}})
	if err != nil {
		t.Fatalf("failed to build client: %v", err)
	}
	af := &AzureFileBackend{
		client:           client,
		multipartUploads: make(map[string]*MultipartUploadMetadata),
		versionedShares:  make(map[string]bool),
		logger:           zap.NewNop(),
	}

	ctx := context.Background()
	_, _ = af.ListBuckets(ctx)
	_, _ = af.HeadBucket(ctx, "bucket")
	_ = af.CreateBucket(ctx, "bucket")
	_ = af.DeleteBucket(ctx, "bucket")
	_, _ = af.PutObject(ctx, "bucket", "file", int64(len("data")), strings.NewReader("data"))
	_, _ = af.CopyObject(ctx, "bucket", "file", "bucket", "file-copy")
	_ = af.DeleteObject(ctx, "bucket", "file")
	if _, _, _, _, err := af.HeadObject(ctx, "bucket", "file"); err != nil {
		t.Logf("HeadObject returned error: %v", err)
	}

	// Multipart lifecycle with minimal parts to cover validation paths
	uploadID, err := af.InitiateMultipartUpload(ctx, "bucket", "multi.bin")
	if err != nil {
		t.Fatalf("initiate multipart failed: %v", err)
	}

	if _, err := af.UploadPart(ctx, "bucket", "multi.bin", uploadID, 1, int64(len("p1")), strings.NewReader("p1")); err != nil {
		t.Fatalf("upload part failed: %v", err)
	}

	parts, err := af.ListParts(ctx, "bucket", "multi.bin", uploadID)
	if err != nil || len(parts) != 1 {
		t.Fatalf("expected one part listed, got %v, err %v", len(parts), err)
	}

	if _, err := af.CompleteMultipartUpload(ctx, "bucket", "multi.bin", uploadID, map[int]string{1: "etag"}); err == nil {
		t.Fatalf("expected complete multipart to fail with fake transport")
	}

	if uploads, err := af.ListMultipartUploads(ctx, "bucket"); err != nil || len(uploads) != 1 {
		t.Fatalf("expected one upload listed, got %d, err %v", len(uploads), err)
	}

	if err := af.AbortMultipartUpload(ctx, "bucket", "multi.bin", uploadID); err != nil {
		t.Fatalf("expected abort to succeed after failed completion, got %v", err)
	}

	// Versioning stubs
	if err := af.EnableVersioning(ctx, "bucket"); err != nil {
		t.Fatalf("enable versioning failed: %v", err)
	}
	if enabled, _ := af.GetVersioning(ctx, "bucket"); !enabled {
		t.Fatalf("expected versioning enabled flag")
	}
	if _, err := af.GetObjectVersion(ctx, "bucket", "file", "v1"); err != nil {
		t.Logf("GetObjectVersion returned error: %v", err)
	}
	if err := af.DeleteObjectVersion(ctx, "bucket", "file", "v1"); err != nil {
		t.Logf("DeleteObjectVersion returned error: %v", err)
	}

	_ = af.Close()
}

// listRangeTransport returns deterministic responses for directory listing, HEAD, and range download requests.
type listRangeTransport struct {
	data string
}

func (listRangeTransport) directoryXML(path string) string {
	if strings.Contains(path, "dir1") {
		return "<?xml version=\"1.0\" encoding=\"utf-8\"?><EnumerationResults><Entries><File><Name>nested.txt</Name></File></Entries><NextMarker></NextMarker></EnumerationResults>"
	}
	return "<?xml version=\"1.0\" encoding=\"utf-8\"?><EnumerationResults><Entries><File><Name>alpha.txt</Name></File><Directory><Name>dir1</Name></Directory></Entries><NextMarker></NextMarker></EnumerationResults>"
}

func (lt listRangeTransport) Do(req *http.Request) (*http.Response, error) { // Implements azcore.Transporter
	q := req.URL.Query()
	headers := http.Header{}
	headers.Set("Date", time.Now().UTC().Format(http.TimeFormat))

	if q.Get("restype") == "directory" && q.Get("comp") == "list" {
		body := lt.directoryXML(req.URL.Path)
		return &http.Response{StatusCode: http.StatusOK, Header: headers, Body: io.NopCloser(strings.NewReader(body))}, nil
	}

	if req.Method == http.MethodHead {
		headers.Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		headers.Set("Content-Length", strconv.Itoa(len(lt.data)))
		return &http.Response{StatusCode: http.StatusOK, Header: headers, Body: http.NoBody}, nil
	}

	body := lt.data
	status := http.StatusOK
	rng := req.Header.Get("Range")
	if rng == "" {
		rng = req.Header.Get("x-ms-range")
	}
	if strings.HasPrefix(rng, "bytes=") {
		parts := strings.Split(strings.TrimPrefix(rng, "bytes="), "-")
		start, _ := strconv.Atoi(parts[0])
		end := len(lt.data) - 1
		if len(parts) > 1 && parts[1] != "" {
			if parsed, err := strconv.Atoi(parts[1]); err == nil {
				end = parsed
			}
		}
		if start < 0 {
			start = 0
		}
		if end >= len(lt.data) {
			end = len(lt.data) - 1
		}
		if start <= end {
			body = lt.data[start : end+1]
		} else {
			body = ""
		}
		headers.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(lt.data)))
		status = http.StatusPartialContent
	}
	headers.Set("Content-Length", strconv.Itoa(len(body)))

	return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body))}, nil
}

func TestAzureFileListAndRange(t *testing.T) {
	client, err := service.NewClientWithNoCredential("https://example.com", &service.ClientOptions{ClientOptions: azcore.ClientOptions{Transport: listRangeTransport{data: "helloworld"}}})
	if err != nil {
		t.Fatalf("failed to build client: %v", err)
	}
	af := &AzureFileBackend{
		client:           client,
		multipartUploads: make(map[string]*MultipartUploadMetadata),
		versionedShares:  make(map[string]bool),
		logger:           zap.NewNop(),
	}

	ctx := context.Background()
	objects, err := af.ListObjects(ctx, "share", "")
	if err != nil || len(objects) != 2 {
		t.Fatalf("expected two objects from listing, got %v err=%v", objects, err)
	}

	page, token, err := af.ListObjectsV2(ctx, "share", "", "", 1)
	if err != nil || len(page) != 1 || token == "" {
		t.Fatalf("expected paginated results with continuation token, got page=%v token=%q err=%v", page, token, err)
	}

	info, err := af.GetObjectRange(ctx, "share", "alpha.txt", 0, 5)
	if err != nil {
		t.Fatalf("GetObjectRange failed: %v", err)
	}
	data, _ := io.ReadAll(info.Body)
	if !strings.HasPrefix(string(data), "hello") {
		t.Fatalf("unexpected range data: %s", string(data))
	}

	versions, err := af.ListObjectVersions(ctx, "share", "")
	if err != nil || len(versions) != len(objects) {
		t.Fatalf("expected versions for each object, got %d err=%v", len(versions), err)
	}
}
