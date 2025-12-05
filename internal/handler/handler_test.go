package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// MockBackend is a simple mock implementation of StorageBackend for testing
type MockBackend struct {
	ListBucketsFunc  func(ctx context.Context) ([]string, error)
	CreateBucketFunc func(ctx context.Context, bucketName string) error
	DeleteBucketFunc func(ctx context.Context, bucketName string) error
	PutObjectFunc    func(ctx context.Context, bucketName, objectKey string, data io.Reader) error
	GetObjectFunc    func(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error)
	DeleteObjectFunc func(ctx context.Context, bucketName, objectKey string) error
	HeadObjectFunc   func(ctx context.Context, bucketName, objectKey string) (bool, error)
	ListObjectsFunc  func(ctx context.Context, bucketName, prefix string) ([]string, error)
}

func (m *MockBackend) ListBuckets(ctx context.Context) ([]string, error) {
	if m.ListBucketsFunc != nil {
		return m.ListBucketsFunc(ctx)
	}
	return []string{"bucket1", "bucket2"}, nil
}

func (m *MockBackend) CreateBucket(ctx context.Context, bucketName string) error {
	if m.CreateBucketFunc != nil {
		return m.CreateBucketFunc(ctx, bucketName)
	}
	return nil
}

func (m *MockBackend) DeleteBucket(ctx context.Context, bucketName string) error {
	if m.DeleteBucketFunc != nil {
		return m.DeleteBucketFunc(ctx, bucketName)
	}
	return nil
}

func (m *MockBackend) PutObject(ctx context.Context, bucketName, objectKey string, data io.Reader) error {
	if m.PutObjectFunc != nil {
		return m.PutObjectFunc(ctx, bucketName, objectKey, data)
	}
	return nil
}

func (m *MockBackend) GetObject(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error) {
	if m.GetObjectFunc != nil {
		return m.GetObjectFunc(ctx, bucketName, objectKey)
	}
	return io.NopCloser(bytes.NewReader([]byte("test data"))), nil
}

func (m *MockBackend) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	if m.DeleteObjectFunc != nil {
		return m.DeleteObjectFunc(ctx, bucketName, objectKey)
	}
	return nil
}

func (m *MockBackend) HeadObject(ctx context.Context, bucketName, objectKey string) (bool, error) {
	if m.HeadObjectFunc != nil {
		return m.HeadObjectFunc(ctx, bucketName, objectKey)
	}
	return true, nil
}

func (m *MockBackend) ListObjects(ctx context.Context, bucketName, prefix string) ([]string, error) {
	if m.ListObjectsFunc != nil {
		return m.ListObjectsFunc(ctx, bucketName, prefix)
	}
	return []string{"key1", "key2"}, nil
}

func TestS3HandlerCreation(t *testing.T) {
	mockBackend := &MockBackend{}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	handler := NewS3Handler(mockBackend, logger)
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestListBucketsHandler(t *testing.T) {
	tests := []struct {
		name           string
		listBucketsErr error
		expectedStatus int
	}{
		{
			name:           "success",
			listBucketsErr: nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "backend error",
			listBucketsErr: errors.New("backend error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackend := &MockBackend{
				ListBucketsFunc: func(ctx context.Context) ([]string, error) {
					return []string{"bucket1"}, tt.listBucketsErr
				},
			}
			logger, _ := zap.NewDevelopment()
			defer logger.Sync()
			handler := NewS3Handler(mockBackend, logger)

			r := chi.NewRouter()
			r.Get("/", handler.ListBucketsHandler)

			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestCreateBucketHandler(t *testing.T) {
	tests := []struct {
		name            string
		createBucketErr error
		expectedStatus  int
	}{
		{
			name:            "success",
			createBucketErr: nil,
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "bucket already exists",
			createBucketErr: errors.New("ContainerAlreadyExists"),
			expectedStatus:  http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackend := &MockBackend{
				CreateBucketFunc: func(ctx context.Context, bucketName string) error {
					return tt.createBucketErr
				},
			}
			logger, _ := zap.NewDevelopment()
			defer logger.Sync()
			handler := NewS3Handler(mockBackend, logger)

			r := chi.NewRouter()
			r.Put("/{bucket}", handler.CreateBucketHandler)

			req := httptest.NewRequest("PUT", "/mybucket", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestDeleteBucketHandler(t *testing.T) {
	tests := []struct {
		name            string
		deleteBucketErr error
		expectedStatus  int
	}{
		{
			name:            "success",
			deleteBucketErr: nil,
			expectedStatus:  http.StatusNoContent,
		},
		{
			name:            "bucket not found",
			deleteBucketErr: errors.New("ContainerNotFound"),
			expectedStatus:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackend := &MockBackend{
				DeleteBucketFunc: func(ctx context.Context, bucketName string) error {
					return tt.deleteBucketErr
				},
			}
			logger, _ := zap.NewDevelopment()
			defer logger.Sync()
			handler := NewS3Handler(mockBackend, logger)

			r := chi.NewRouter()
			r.Delete("/{bucket}", handler.DeleteBucketHandler)

			req := httptest.NewRequest("DELETE", "/mybucket", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestListObjectsV2Handler(t *testing.T) {
	mockBackend := &MockBackend{
		ListObjectsFunc: func(ctx context.Context, bucketName, prefix string) ([]string, error) {
			return []string{"obj1", "obj2"}, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}", handler.ListObjectsV2Handler)

	req := httptest.NewRequest("GET", "/mybucket", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestPutObjectHandler(t *testing.T) {
	mockBackend := &MockBackend{
		PutObjectFunc: func(ctx context.Context, bucketName, objectKey string, data io.Reader) error {
			return nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Put("/{bucket}/{key}", handler.PutObjectHandler)

	body := bytes.NewReader([]byte("test data"))
	req := httptest.NewRequest("PUT", "/mybucket/mykey", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGetObjectHandler(t *testing.T) {
	mockBackend := &MockBackend{
		GetObjectFunc: func(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("test data"))), nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}/{key}", handler.GetObjectHandler)

	req := httptest.NewRequest("GET", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHeadObjectHandler(t *testing.T) {
	mockBackend := &MockBackend{
		HeadObjectFunc: func(ctx context.Context, bucketName, objectKey string) (bool, error) {
			return true, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Head("/{bucket}/{key}", handler.HeadObjectHandler)

	req := httptest.NewRequest("HEAD", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestDeleteObjectHandler(t *testing.T) {
	mockBackend := &MockBackend{
		DeleteObjectFunc: func(ctx context.Context, bucketName, objectKey string) error {
			return nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Delete("/{bucket}/{key}", handler.DeleteObjectHandler)

	req := httptest.NewRequest("DELETE", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}
