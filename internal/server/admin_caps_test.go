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
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/auth"
	"github.com/vibhansa-msft/azs3-proxy/internal/backend"
	"github.com/vibhansa-msft/azs3-proxy/internal/config"
)

// dummyBackend implements StorageBackend and BandwidthUpdater for testing admin caps route.
type dummyBackend struct{ last [3]float64 }

func (d *dummyBackend) UpdateCaps(r, w, c float64) { d.last = [3]float64{r, w, c} }

func (d *dummyBackend) ListBuckets(context.Context) ([]string, error)    { return nil, nil }
func (d *dummyBackend) CreateBucket(context.Context, string) error       { return nil }
func (d *dummyBackend) DeleteBucket(context.Context, string) error       { return nil }
func (d *dummyBackend) HeadBucket(context.Context, string) (bool, error) { return true, nil }
func (d *dummyBackend) PutObject(context.Context, string, string, int64, io.Reader) (string, error) {
	return "", nil
}
func (d *dummyBackend) CopyObject(context.Context, string, string, string, string) (string, error) {
	return "", nil
}
func (d *dummyBackend) GetObject(context.Context, string, string) (backend.ObjectInfo, error) {
	return backend.ObjectInfo{}, nil
}
func (d *dummyBackend) GetObjectRange(context.Context, string, string, int64, int64) (backend.ObjectInfo, error) {
	return backend.ObjectInfo{}, nil
}
func (d *dummyBackend) DeleteObject(context.Context, string, string) error { return nil }
func (d *dummyBackend) HeadObject(context.Context, string, string) (bool, int64, time.Time, error) {
	return true, 0, time.Time{}, nil
}
func (d *dummyBackend) ListObjects(context.Context, string, string) ([]backend.ObjectListItem, error) {
	return nil, nil
}
func (d *dummyBackend) ListObjectsV2(context.Context, string, string, string, int32) ([]backend.ObjectListItem, string, error) {
	return nil, "", nil
}
func (d *dummyBackend) InitiateMultipartUpload(context.Context, string, string) (string, error) {
	return "", nil
}
func (d *dummyBackend) UploadPart(context.Context, string, string, string, int, int64, io.Reader) (string, error) {
	return "", nil
}
func (d *dummyBackend) CompleteMultipartUpload(context.Context, string, string, string, map[int]string) (string, error) {
	return "", nil
}
func (d *dummyBackend) AbortMultipartUpload(context.Context, string, string, string) error {
	return nil
}
func (d *dummyBackend) ListParts(context.Context, string, string, string) ([]interface{}, error) {
	return nil, nil
}
func (d *dummyBackend) ListMultipartUploads(context.Context, string) ([]interface{}, error) {
	return nil, nil
}
func (d *dummyBackend) EnableVersioning(context.Context, string) error      { return nil }
func (d *dummyBackend) GetVersioning(context.Context, string) (bool, error) { return false, nil }
func (d *dummyBackend) ListObjectVersions(context.Context, string, string) ([]interface{}, error) {
	return nil, nil
}
func (d *dummyBackend) GetObjectVersion(context.Context, string, string, string) (backend.ObjectInfo, error) {
	return backend.ObjectInfo{}, nil
}
func (d *dummyBackend) DeleteObjectVersion(context.Context, string, string, string) error { return nil }
func (d *dummyBackend) Close() error                                                      { return nil }

type storageOnlyBackend struct{}

func (storageOnlyBackend) ListBuckets(context.Context) ([]string, error)    { return nil, nil }
func (storageOnlyBackend) CreateBucket(context.Context, string) error       { return nil }
func (storageOnlyBackend) DeleteBucket(context.Context, string) error       { return nil }
func (storageOnlyBackend) HeadBucket(context.Context, string) (bool, error) { return true, nil }
func (storageOnlyBackend) PutObject(context.Context, string, string, int64, io.Reader) (string, error) {
	return "", nil
}
func (storageOnlyBackend) CopyObject(context.Context, string, string, string, string) (string, error) {
	return "", nil
}
func (storageOnlyBackend) GetObject(context.Context, string, string) (backend.ObjectInfo, error) {
	return backend.ObjectInfo{}, nil
}
func (storageOnlyBackend) GetObjectRange(context.Context, string, string, int64, int64) (backend.ObjectInfo, error) {
	return backend.ObjectInfo{}, nil
}
func (storageOnlyBackend) DeleteObject(context.Context, string, string) error { return nil }
func (storageOnlyBackend) HeadObject(context.Context, string, string) (bool, int64, time.Time, error) {
	return true, 0, time.Time{}, nil
}
func (storageOnlyBackend) ListObjects(context.Context, string, string) ([]backend.ObjectListItem, error) {
	return nil, nil
}
func (storageOnlyBackend) ListObjectsV2(context.Context, string, string, string, int32) ([]backend.ObjectListItem, string, error) {
	return nil, "", nil
}
func (storageOnlyBackend) InitiateMultipartUpload(context.Context, string, string) (string, error) {
	return "", nil
}
func (storageOnlyBackend) UploadPart(context.Context, string, string, string, int, int64, io.Reader) (string, error) {
	return "", nil
}
func (storageOnlyBackend) CompleteMultipartUpload(context.Context, string, string, string, map[int]string) (string, error) {
	return "", nil
}
func (storageOnlyBackend) AbortMultipartUpload(context.Context, string, string, string) error {
	return nil
}
func (storageOnlyBackend) ListParts(context.Context, string, string, string) ([]interface{}, error) {
	return nil, nil
}
func (storageOnlyBackend) ListMultipartUploads(context.Context, string) ([]interface{}, error) {
	return nil, nil
}
func (storageOnlyBackend) EnableVersioning(context.Context, string) error      { return nil }
func (storageOnlyBackend) GetVersioning(context.Context, string) (bool, error) { return false, nil }
func (storageOnlyBackend) ListObjectVersions(context.Context, string, string) ([]interface{}, error) {
	return nil, nil
}
func (storageOnlyBackend) GetObjectVersion(context.Context, string, string, string) (backend.ObjectInfo, error) {
	return backend.ObjectInfo{}, nil
}
func (storageOnlyBackend) DeleteObjectVersion(context.Context, string, string, string) error {
	return nil
}
func (storageOnlyBackend) Close() error { return nil }

func TestAdminCapsUpdateAuthorized(t *testing.T) {
	cfg := &config.Config{
		S3AccessKeyID:     "id",
		S3SecretAccessKey: "secret",
		AzureAuth: &config.AzureAuthConfig{
			Mode:               config.AuthModeAccountKey,
			StorageAccountName: "acct",
			AccountKey:         "dGVzdA==",
		},
		AdminToken: "tok",
	}

	backend := &dummyBackend{}
	server := &S3ProxyServer{
		router:  chi.NewRouter(),
		config:  cfg,
		logger:  zap.NewNop(),
		backend: backend,
		auth:    auth.NewAuthVerifier(cfg.S3AccessKeyID, cfg.S3SecretAccessKey),
	}

	server.registerMiddleware()
	server.registerRoutes()

	req := httptest.NewRequest(http.MethodPut, "/admin/caps", strings.NewReader(`{"read_mbps":5,"write_mbps":0,"combined_mbps":0}`))
	req.Header.Set("X-Admin-Token", "tok")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if backend.last[0] != 5 {
		t.Fatalf("expected read cap 5, got %v", backend.last[0])
	}
}

func TestAdminCapsUpdateUnauthorized(t *testing.T) {
	cfg := &config.Config{
		S3AccessKeyID:     "id",
		S3SecretAccessKey: "secret",
		AzureAuth: &config.AzureAuthConfig{
			Mode:               config.AuthModeAccountKey,
			StorageAccountName: "acct",
			AccountKey:         "dGVzdA==",
		},
		AdminToken: "tok",
	}

	backend := &dummyBackend{}
	server := &S3ProxyServer{
		router:  chi.NewRouter(),
		config:  cfg,
		logger:  zap.NewNop(),
		backend: backend,
		auth:    auth.NewAuthVerifier(cfg.S3AccessKeyID, cfg.S3SecretAccessKey),
	}

	server.registerMiddleware()
	server.registerRoutes()

	req := httptest.NewRequest(http.MethodPut, "/admin/caps", strings.NewReader(`{"read_mbps":1,"write_mbps":0,"combined_mbps":0}`))
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when token missing, got %d", w.Code)
	}
	if backend.last[0] != 0 {
		t.Fatalf("expected cap unchanged, got %v", backend.last[0])
	}
}

func TestAdminCapsInvalidJSON(t *testing.T) {
	cfg := &config.Config{
		S3AccessKeyID:     "id",
		S3SecretAccessKey: "secret",
		AzureAuth: &config.AzureAuthConfig{
			Mode:               config.AuthModeAccountKey,
			StorageAccountName: "acct",
			AccountKey:         "dGVzdA==",
		},
		AdminToken: "tok",
	}

	backend := &dummyBackend{}
	server := &S3ProxyServer{
		router:  chi.NewRouter(),
		config:  cfg,
		logger:  zap.NewNop(),
		backend: backend,
		auth:    auth.NewAuthVerifier(cfg.S3AccessKeyID, cfg.S3SecretAccessKey),
	}

	server.registerMiddleware()
	server.registerRoutes()

	req := httptest.NewRequest(http.MethodPut, "/admin/caps", strings.NewReader("not-json"))
	req.Header.Set("X-Admin-Token", "tok")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d", w.Code)
	}
}

func TestAdminCapsValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"negative", `{"read_mbps":-1}`},
		{"conflict", `{"read_mbps":1,"combined_mbps":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				S3AccessKeyID:     "id",
				S3SecretAccessKey: "secret",
				AzureAuth: &config.AzureAuthConfig{
					Mode:               config.AuthModeAccountKey,
					StorageAccountName: "acct",
					AccountKey:         "dGVzdA==",
				},
				AdminToken: "tok",
			}

			backend := &dummyBackend{}
			server := &S3ProxyServer{
				router:  chi.NewRouter(),
				config:  cfg,
				logger:  zap.NewNop(),
				backend: backend,
				auth:    auth.NewAuthVerifier(cfg.S3AccessKeyID, cfg.S3SecretAccessKey),
			}

			server.registerMiddleware()
			server.registerRoutes()

			req := httptest.NewRequest(http.MethodPut, "/admin/caps", strings.NewReader(tt.body))
			req.Header.Set("X-Admin-Token", "tok")
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestAdminCapsBackendWithoutUpdater(t *testing.T) {
	cfg := &config.Config{
		S3AccessKeyID:     "id",
		S3SecretAccessKey: "secret",
		AzureAuth: &config.AzureAuthConfig{
			Mode:               config.AuthModeAccountKey,
			StorageAccountName: "acct",
			AccountKey:         "dGVzdA==",
		},
		AdminToken: "tok",
	}

	server := &S3ProxyServer{
		router:  chi.NewRouter(),
		config:  cfg,
		logger:  zap.NewNop(),
		backend: storageOnlyBackend{},
		auth:    auth.NewAuthVerifier(cfg.S3AccessKeyID, cfg.S3SecretAccessKey),
	}

	server.registerMiddleware()
	server.registerRoutes()

	req := httptest.NewRequest(http.MethodPut, "/admin/caps", strings.NewReader(`{"read_mbps":1}`))
	req.Header.Set("X-Admin-Token", "tok")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", w.Code)
	}
}

func TestAdminCapsNoTokenConfigured(t *testing.T) {
	cfg := &config.Config{
		S3AccessKeyID:     "id",
		S3SecretAccessKey: "secret",
		AzureAuth: &config.AzureAuthConfig{
			Mode:               config.AuthModeAccountKey,
			StorageAccountName: "acct",
			AccountKey:         "dGVzdA==",
		},
		AdminToken: "",
	}

	backend := &dummyBackend{}
	server := &S3ProxyServer{
		router:  chi.NewRouter(),
		config:  cfg,
		logger:  zap.NewNop(),
		backend: backend,
		auth:    auth.NewAuthVerifier(cfg.S3AccessKeyID, cfg.S3SecretAccessKey),
	}

	server.registerMiddleware()
	server.registerRoutes()

	req := httptest.NewRequest(http.MethodPut, "/admin/caps", strings.NewReader(`{"read_mbps":2}`))
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 without token when none configured, got %d", w.Code)
	}
	if backend.last[0] != 2 {
		t.Fatalf("expected read cap 2, got %v", backend.last[0])
	}
}
