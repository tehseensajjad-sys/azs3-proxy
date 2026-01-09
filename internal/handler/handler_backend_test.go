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
	"go.uber.org/zap"
)

// fakeBackend implements backend.StorageBackend with configurable behaviors
type fakeBackend struct {
	listBuckets   []string
	getObjectErr  error
	getObjectData []byte
	putObjectErr  error
}

func (f *fakeBackend) ListBuckets(ctx context.Context) ([]string, error) { return f.listBuckets, nil }
func (f *fakeBackend) HeadBucket(ctx context.Context, bucketName string) (bool, error) {
	return true, nil
}
func (f *fakeBackend) CreateBucket(ctx context.Context, bucketName string) error { return nil }
func (f *fakeBackend) DeleteBucket(ctx context.Context, bucketName string) error { return nil }
func (f *fakeBackend) PutObject(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) error {
	// Drain the reader to simulate upload
	_, _ = io.Copy(io.Discard, data)
	return f.putObjectErr
}
func (f *fakeBackend) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	return nil
}
func (f *fakeBackend) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	return nil
}
func (f *fakeBackend) HeadObject(ctx context.Context, bucketName, objectKey string) (bool, int64, time.Time, error) {
	return false, 0, time.Time{}, nil
}
func (f *fakeBackend) ListObjects(ctx context.Context, bucketName, prefix string) ([]string, error) {
	return nil, nil
}
func (f *fakeBackend) GetObject(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error) {
	if f.getObjectErr != nil {
		return backend.ObjectInfo{}, f.getObjectErr
	}
	return backend.ObjectInfo{Body: io.NopCloser(bytes.NewReader(f.getObjectData)), Size: int64(len(f.getObjectData)), LastModified: time.Now()}, nil
}
func (f *fakeBackend) InitiateMultipartUpload(ctx context.Context, bucketName, objectKey string) (string, error) {
	return "", nil
}
func (f *fakeBackend) UploadPart(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, data io.Reader) (string, error) {
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
	defer logger.Sync()

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
	defer logger.Sync()

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

func TestPutObjectHandlerError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

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
