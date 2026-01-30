package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/auth"
	"github.com/vibhansa-msft/azs3-proxy/internal/backend"
	"github.com/vibhansa-msft/azs3-proxy/internal/config"
)

type stubBackend struct{}

func (stubBackend) ListBuckets(ctx context.Context) ([]string, error)               { return []string{"bucket"}, nil }
func (stubBackend) CreateBucket(ctx context.Context, bucketName string) error       { return nil }
func (stubBackend) DeleteBucket(ctx context.Context, bucketName string) error       { return nil }
func (stubBackend) HeadBucket(ctx context.Context, bucketName string) (bool, error) { return true, nil }
func (stubBackend) PutObject(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) error {
	return nil
}
func (stubBackend) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	return nil
}
func (stubBackend) GetObject(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error) {
	return backend.ObjectInfo{Body: io.NopCloser(strings.NewReader("ok")), LastModified: time.Now(), Size: 2}, nil
}
func (stubBackend) DeleteObject(ctx context.Context, bucketName, objectKey string) error { return nil }
func (stubBackend) HeadObject(ctx context.Context, bucketName, objectKey string) (bool, int64, time.Time, error) {
	return true, 1, time.Now(), nil
}
func (stubBackend) ListObjects(ctx context.Context, bucketName, prefix string) ([]string, error) {
	return []string{"obj"}, nil
}
func (stubBackend) ListObjectsV2(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) ([]string, string, error) {
	objs, _ := (stubBackend{}).ListObjects(ctx, bucketName, prefix)
	return objs, "", nil
}
func (stubBackend) InitiateMultipartUpload(ctx context.Context, bucketName, objectKey string) (string, error) {
	return "upload", nil
}
func (stubBackend) UploadPart(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, size int64, data io.Reader) (string, error) {
	return "etag", nil
}
func (stubBackend) CompleteMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string, partETags map[int]string) (string, error) {
	return "etag", nil
}
func (stubBackend) AbortMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string) error {
	return nil
}
func (stubBackend) ListParts(ctx context.Context, bucketName, objectKey, uploadID string) ([]interface{}, error) {
	return []interface{}{map[string]interface{}{"PartNumber": 1}}, nil
}
func (stubBackend) ListMultipartUploads(ctx context.Context, bucketName string) ([]interface{}, error) {
	return []interface{}{map[string]interface{}{"UploadID": "upload"}}, nil
}
func (stubBackend) EnableVersioning(ctx context.Context, bucketName string) error { return nil }
func (stubBackend) GetVersioning(ctx context.Context, bucketName string) (bool, error) {
	return true, nil
}
func (stubBackend) ListObjectVersions(ctx context.Context, bucketName, prefix string) ([]interface{}, error) {
	return []interface{}{map[string]interface{}{"VersionID": "v1"}}, nil
}
func (stubBackend) GetObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) (backend.ObjectInfo, error) {
	return backend.ObjectInfo{Body: io.NopCloser(strings.NewReader("v1")), LastModified: time.Now(), Size: 2}, nil
}
func (stubBackend) DeleteObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) error {
	return nil
}
func (stubBackend) Close() error { return nil }

func TestRequestIDMiddlewareUsesHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Context().Value(middleware.RequestIDKey); got == nil {
			t.Fatalf("expected request id in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := requestIDMiddleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(middleware.RequestIDHeader, "req-123")
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	if got := rr.Header().Get(middleware.RequestIDHeader); got != "req-123" {
		t.Fatalf("expected header propagated, got %s", got)
	}
}

func TestRegisterMiddlewareWithDebugResponses(t *testing.T) {
	t.Setenv("DEBUG_RESPONSES", "true")
	logger := zap.NewNop()
	router := chi.NewRouter()
	s := &S3ProxyServer{
		router: router,
		logger: logger,
		auth:   auth.NewAuthVerifier("id", "secret"),
	}
	s.registerMiddleware()

	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected status 418, got %d", rr.Code)
	}
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected X-Request-ID header set")
	}
}

func TestRegisterRoutesWithCacheEnabled(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.Config{
		CacheEnabled: true,
		CachePath:    tmp,
		CacheMaxSize: 1024,
		CacheTTL:     60,
	}
	router := chi.NewRouter()
	s := &S3ProxyServer{
		router:  router,
		config:  cfg,
		logger:  zap.NewNop(),
		backend: stubBackend{},
	}

	s.registerRoutes()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected health check 200, got %d", rr.Code)
	}
}
