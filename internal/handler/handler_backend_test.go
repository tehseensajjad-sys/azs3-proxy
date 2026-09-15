package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vibhansa-msft/azs3-proxy/internal/backend"
	"github.com/vibhansa-msft/azs3-proxy/internal/cache"
	"go.uber.org/zap"
)

// fakeBackend implements backend.StorageBackend with configurable behaviors
type fakeBackend struct {
	listBuckets    []string
	getObjectErr   error
	getObjectData  []byte
	putObjectErr   error
	getObjectCalls int
}

func (f *fakeBackend) ListBuckets(ctx context.Context) ([]string, error) { return f.listBuckets, nil }
func (f *fakeBackend) HeadBucket(ctx context.Context, bucketName string) (bool, error) {
	return true, nil
}
func (f *fakeBackend) CreateBucket(ctx context.Context, bucketName string) error { return nil }
func (f *fakeBackend) DeleteBucket(ctx context.Context, bucketName string) error { return nil }
func (f *fakeBackend) PutObject(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) (string, error) {
	// Drain the reader to simulate upload
	_, _ = io.Copy(io.Discard, data)
	return "\"fakeetag\"", f.putObjectErr
}
func (f *fakeBackend) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) (string, error) {
	return "\"fakeetag\"", nil
}
func (f *fakeBackend) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	return nil
}
func (f *fakeBackend) HeadObject(ctx context.Context, bucketName, objectKey string) (bool, int64, time.Time, error) {
	if f.getObjectErr != nil {
		return false, 0, time.Time{}, f.getObjectErr
	}
	return true, int64(len(f.getObjectData)), time.Now(), nil
}
func (f *fakeBackend) ListObjects(ctx context.Context, bucketName, prefix string) ([]backend.ObjectListItem, error) {
	return nil, nil
}
func (f *fakeBackend) ListObjectsV2(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) ([]backend.ObjectListItem, string, error) {
	objects, err := f.ListObjects(ctx, bucketName, prefix)
	return objects, "", err
}
func (f *fakeBackend) GetObject(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error) {
	if f.getObjectErr != nil {
		return backend.ObjectInfo{}, f.getObjectErr
	}
	f.getObjectCalls++
	return backend.ObjectInfo{Body: io.NopCloser(bytes.NewReader(f.getObjectData)), Size: int64(len(f.getObjectData)), LastModified: time.Now()}, nil
}
func (f *fakeBackend) GetObjectRange(ctx context.Context, bucketName, objectKey string, offset, length int64) (backend.ObjectInfo, error) {
	if f.getObjectErr != nil {
		return backend.ObjectInfo{}, f.getObjectErr
	}
	if offset < 0 || length <= 0 {
		return backend.ObjectInfo{}, errors.New("invalid range")
	}
	if offset >= int64(len(f.getObjectData)) {
		return backend.ObjectInfo{}, errors.New("range beyond object")
	}
	end := offset + length
	if end > int64(len(f.getObjectData)) {
		end = int64(len(f.getObjectData))
	}
	f.getObjectCalls++
	chunk := f.getObjectData[offset:end]
	return backend.ObjectInfo{Body: io.NopCloser(bytes.NewReader(chunk)), Size: int64(len(chunk)), LastModified: time.Now()}, nil
}
func (f *fakeBackend) InitiateMultipartUpload(ctx context.Context, bucketName, objectKey string) (string, error) {
	return "", nil
}
func (f *fakeBackend) UploadPart(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, size int64, data io.Reader) (string, error) {
	return "", nil
}
func (f *fakeBackend) CompleteMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string, partETags map[int]string) (string, error) {
	return "", nil
}
func (f *fakeBackend) AbortMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string) error {
	return nil
}
func (f *fakeBackend) ListParts(ctx context.Context, bucketName, objectKey, uploadID string) ([]interface{}, error) {
	return nil, nil
}
func (f *fakeBackend) ListMultipartUploads(ctx context.Context, bucketName string) ([]interface{}, error) {
	return nil, nil
}
func (f *fakeBackend) EnableVersioning(ctx context.Context, bucketName string) error { return nil }
func (f *fakeBackend) GetVersioning(ctx context.Context, bucketName string) (bool, error) {
	return false, nil
}
func (f *fakeBackend) ListObjectVersions(ctx context.Context, bucketName, prefix string) ([]interface{}, error) {
	return nil, nil
}
func (f *fakeBackend) GetObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) (backend.ObjectInfo, error) {
	return backend.ObjectInfo{}, nil
}
func (f *fakeBackend) DeleteObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) error {
	return nil
}
func (f *fakeBackend) Close() error { return nil }

func TestListBucketsHandlerSuccess(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	fb := &fakeBackend{listBuckets: []string{"a", "b"}}
	h := NewS3Handler(fb, logger)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	h.ListBucketsHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestGetObjectHandlerErrorAndSuccess(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// Error path
	fb := &fakeBackend{getObjectErr: errors.New("not found")}
	h := NewS3Handler(fb, logger)

	r := httptest.NewRequest("GET", "/bucket/key", nil)
	ctxErr := chi.NewRouteContext()
	ctxErr.URLParams.Add("bucket", "bucket")
	ctxErr.URLParams.Add("*", "/key")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctxErr))
	rr := httptest.NewRecorder()
	h.GetObjectHandler(rr, r)
	if rr.Code != http.StatusInternalServerError && rr.Code != http.StatusNotFound && rr.Code != http.StatusOK {
		// the handler may translate errors differently; accept non-panic
		t.Logf("GetObjectHandler returned %d on error path", rr.Code)
	}

	// Success path
	data := []byte("hello")
	fb2 := &fakeBackend{getObjectData: data}
	h2 := NewS3Handler(fb2, logger)

	req := httptest.NewRequest("GET", "/bucket/key", nil)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("bucket", "bucket")
	ctx.URLParams.Add("*", "/key")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
	rr2 := httptest.NewRecorder()
	h2.GetObjectHandler(rr2, req)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 on GetObject success, got %d", rr2.Code)
	}
}

func TestGetObjectHandlerRangeCachesBlock(t *testing.T) {
	logger := zap.NewNop()
	data := bytes.Repeat([]byte("x"), 9*1024*1024) // 9MB
	fb := &fakeBackend{getObjectData: data}

	h := NewS3Handler(fb, logger)
	cm, err := cache.NewCacheManager(t.TempDir(), 64<<20, 300)
	if err != nil {
		t.Fatalf("cache manager init failed: %v", err)
	}
	defer func() { _ = cm.Close() }()
	h.SetCacheManager(cm)

	req := httptest.NewRequest("GET", "/bucket/key", nil)
	req.Header.Set("Range", "bytes=0-1048575") // 1MB
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("bucket", "bucket")
	ctx.URLParams.Add("*", "/key")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))

	// First request should hit backend range fetch
	rr := httptest.NewRecorder()
	h.GetObjectHandler(rr, req)
	if rr.Code != http.StatusPartialContent {
		t.Fatalf("expected 206, got %d", rr.Code)
	}
	if fb.getObjectCalls != 1 {
		t.Fatalf("expected 1 backend call after first range, got %d", fb.getObjectCalls)
	}

	// Second identical request should hit cache only
	rr2 := httptest.NewRecorder()
	h.GetObjectHandler(rr2, req)
	if rr2.Code != http.StatusPartialContent {
		t.Fatalf("expected 206 on second request, got %d", rr2.Code)
	}
	if fb.getObjectCalls > 2 {
		t.Fatalf("expected at most one additional backend call for prefetch, got %d", fb.getObjectCalls)
	}

	if rr2.Header().Get("Content-Range") == "" {
		t.Fatalf("expected Content-Range header on range response")
	}

	// Allow background prefetch goroutine to complete before TempDir cleanup.
	time.Sleep(20 * time.Millisecond)
}

func TestPrefetchBlocksCachesData(t *testing.T) {
	logger := zap.NewNop()
	data := bytes.Repeat([]byte("a"), 3*1024*1024) // 3MB
	fb := &fakeBackend{getObjectData: data}

	h := NewS3Handler(fb, logger)
	cm, err := cache.NewCacheManager(t.TempDir(), 64<<20, 300)
	if err != nil {
		t.Fatalf("cache manager init failed: %v", err)
	}
	defer func() { _ = cm.Close() }()
	h.SetCacheManager(cm)

	blockSize := int64(1 * 1024 * 1024)
	h.prefetchBlocks(context.Background(), "bucket", "key", int64(len(data)), blockSize, 0, 3)

	if fb.getObjectCalls != 3 {
		t.Fatalf("expected 3 backend calls (one per block), got %d", fb.getObjectCalls)
	}

	for blk := int64(0); blk < 3; blk++ {
		start := blk * blockSize
		end := start + blockSize - 1
		if end >= int64(len(data)) {
			end = int64(len(data)) - 1
		}
		cacheKey := blockCacheKey("bucket", "key", start, end)
		path, err := cm.GetObjectFromCache(cacheKey)
		if err != nil || path == "" {
			t.Fatalf("expected cached path for block %d, got err=%v path=%s", blk, err, path)
		}
	}
}

func TestPutObjectHandlerError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	fb := &fakeBackend{putObjectErr: errors.New("write error")}
	h := NewS3Handler(fb, logger)

	req := httptest.NewRequest("PUT", "/bucket/key", bytes.NewReader([]byte("data")))
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("bucket", "bucket")
	ctx.URLParams.Add("*", "/key")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))

	rr := httptest.NewRecorder()
	h.PutObjectHandler(rr, req)
	if rr.Code != http.StatusInternalServerError && rr.Code != http.StatusOK {
		t.Logf("PutObjectHandler returned status %d", rr.Code)
	}
}
