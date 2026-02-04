package handler

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vibhansa-msft/azs3-proxy/internal/backend"
	"github.com/vibhansa-msft/azs3-proxy/internal/cache"
	"github.com/vibhansa-msft/azs3-proxy/internal/models"
	"github.com/vibhansa-msft/azs3-proxy/internal/telemetry"
	"go.uber.org/zap"
)

func TestSetTelemetryManagerAndWriteErrorResponse(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	h := NewS3Handler(nil, logger)
	if h == nil {
		t.Fatal("handler is nil")
	}

	// Set a telemetry manager stub (nil metrics provider is acceptable)
	tel, _ := telemetry.NewManager(context.Background())
	if tel == nil {
		h.SetTelemetryManager(nil)
	} else {
		h.SetTelemetryManager(tel)
	}
	h.SetTelemetryManager(tel)

	// Exercise writeErrorResponse to ensure XML is written and no panic
	rr := httptest.NewRecorder()
	s3err := &models.S3Error{Code: models.InternalError, Message: "boom", Resource: "/bucket/key"}
	h.writeErrorResponse(rr, s3err)
	if rr.Result().StatusCode == 0 {
		t.Fatalf("expected status code set, got 0")
	}
}

func TestParseSingleRange(t *testing.T) {
	tests := []struct {
		name           string
		header         string
		size           int64
		wantStart      int64
		wantEnd        int64
		expectErr      bool
		expectMultiErr bool
	}{
		{
			name:      "bounded range",
			header:    "bytes=0-9",
			size:      100,
			wantStart: 0,
			wantEnd:   9,
		},
		{
			name:      "suffix range",
			header:    "bytes=-10",
			size:      50,
			wantStart: 40,
			wantEnd:   49,
		},
		{
			name:      "open ended",
			header:    "bytes=5-",
			size:      10,
			wantStart: 5,
			wantEnd:   9,
		},
		{
			name:           "multiple ranges",
			header:         "bytes=0-1,2-3",
			size:           10,
			expectErr:      true,
			expectMultiErr: true,
		},
		{
			name:      "invalid unit",
			header:    "items=0-1",
			size:      10,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := parseSingleRange(tt.header, tt.size)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if tt.expectMultiErr && !errors.Is(err, errMultipleRanges) {
					t.Fatalf("expected multiple ranges error, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("unexpected range: start=%d end=%d", start, end)
			}
		})
	}
}

// MockBackend is a simple mock implementation of StorageBackend for testing
type MockBackend struct {
	ListBucketsFunc             func(ctx context.Context) ([]string, error)
	CreateBucketFunc            func(ctx context.Context, bucketName string) error
	DeleteBucketFunc            func(ctx context.Context, bucketName string) error
	PutObjectFunc               func(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) error
	CopyObjectFunc              func(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error
	GetObjectFunc               func(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error)
	GetObjectRangeFunc          func(ctx context.Context, bucketName, objectKey string, offset, length int64) (backend.ObjectInfo, error)
	DeleteObjectFunc            func(ctx context.Context, bucketName, objectKey string) error
	HeadObjectFunc              func(ctx context.Context, bucketName, objectKey string) (bool, int64, time.Time, error)
	ListObjectsFunc             func(ctx context.Context, bucketName, prefix string) ([]string, error)
	ListObjectsV2Func           func(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) ([]string, string, error)
	InitiateMultipartUploadFunc func(ctx context.Context, bucketName, objectKey string) (string, error)
	UploadPartFunc              func(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, size int64, data io.Reader) (string, error)
	CompleteMultipartUploadFunc func(ctx context.Context, bucketName, objectKey, uploadID string, partETags map[int]string) (string, error)
	AbortMultipartUploadFunc    func(ctx context.Context, bucketName, objectKey, uploadID string) error
	ListPartsFunc               func(ctx context.Context, bucketName, objectKey, uploadID string) ([]interface{}, error)
	ListMultipartUploadsFunc    func(ctx context.Context, bucketName string) ([]interface{}, error)
	EnableVersioningFunc        func(ctx context.Context, bucketName string) error
	GetVersioningFunc           func(ctx context.Context, bucketName string) (bool, error)
	ListObjectVersionsFunc      func(ctx context.Context, bucketName, prefix string) ([]interface{}, error)
	GetObjectVersionFunc        func(ctx context.Context, bucketName, objectKey, versionID string) (backend.ObjectInfo, error)
	DeleteObjectVersionFunc     func(ctx context.Context, bucketName, objectKey, versionID string) error
	HeadBucketFunc              func(ctx context.Context, bucketName string) (bool, error)
	CloseFunc                   func() error
}

func (m *MockBackend) HeadBucket(ctx context.Context, bucketName string) (bool, error) {
	if m.HeadBucketFunc != nil {
		return m.HeadBucketFunc(ctx, bucketName)
	}
	return true, nil
}

func (m *MockBackend) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
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

func (m *MockBackend) PutObject(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) error {
	if m.PutObjectFunc != nil {
		return m.PutObjectFunc(ctx, bucketName, objectKey, size, data)
	}
	return nil
}

func (m *MockBackend) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	if m.CopyObjectFunc != nil {
		return m.CopyObjectFunc(ctx, srcBucket, srcKey, destBucket, destKey)
	}
	return nil
}

func (m *MockBackend) GetObject(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error) {
	if m.GetObjectFunc != nil {
		return m.GetObjectFunc(ctx, bucketName, objectKey)
	}
	return backend.ObjectInfo{Body: io.NopCloser(strings.NewReader("")), Size: 0, LastModified: time.Now()}, nil
}

func (m *MockBackend) GetObjectRange(ctx context.Context, bucketName, objectKey string, offset, length int64) (backend.ObjectInfo, error) {
	if m.GetObjectRangeFunc != nil {
		return m.GetObjectRangeFunc(ctx, bucketName, objectKey, offset, length)
	}
	if m.GetObjectFunc != nil {
		obj, err := m.GetObjectFunc(ctx, bucketName, objectKey)
		if err != nil {
			return obj, err
		}
		data, err := io.ReadAll(obj.Body)
		_ = obj.Body.Close()
		if err != nil {
			return backend.ObjectInfo{}, err
		}
		if offset < 0 {
			offset = 0
		}
		if offset > int64(len(data)) {
			return backend.ObjectInfo{}, errors.New("range start beyond size")
		}
		if length <= 0 || offset+length > int64(len(data)) {
			length = int64(len(data)) - offset
		}
		end := offset + length
		slice := data[offset:end]
		return backend.ObjectInfo{Body: io.NopCloser(bytes.NewReader(slice)), Size: int64(len(data)), LastModified: obj.LastModified}, nil
	}
	return backend.ObjectInfo{Body: io.NopCloser(strings.NewReader("")), Size: 0, LastModified: time.Now()}, nil
}

func (m *MockBackend) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	if m.DeleteObjectFunc != nil {
		return m.DeleteObjectFunc(ctx, bucketName, objectKey)
	}
	return nil
}

func (m *MockBackend) HeadObject(ctx context.Context, bucketName, objectKey string) (bool, int64, time.Time, error) {
	if m.HeadObjectFunc != nil {
		return m.HeadObjectFunc(ctx, bucketName, objectKey)
	}
	return true, int64(len("test data")), time.Now(), nil
}

func (m *MockBackend) ListObjects(ctx context.Context, bucketName, prefix string) ([]string, error) {
	if m.ListObjectsFunc != nil {
		return m.ListObjectsFunc(ctx, bucketName, prefix)
	}
	return []string{"key1", "key2"}, nil
}

func (m *MockBackend) ListObjectsV2(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) ([]string, string, error) {
	if m.ListObjectsV2Func != nil {
		return m.ListObjectsV2Func(ctx, bucketName, prefix, continuationToken, maxResults)
	}
	objects, _ := m.ListObjects(ctx, bucketName, prefix)
	return objects, "", nil
}

func (m *MockBackend) InitiateMultipartUpload(ctx context.Context, bucketName, objectKey string) (string, error) {
	if m.InitiateMultipartUploadFunc != nil {
		return m.InitiateMultipartUploadFunc(ctx, bucketName, objectKey)
	}
	return "test-upload-id", nil
}

func (m *MockBackend) UploadPart(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, size int64, data io.Reader) (string, error) {
	if m.UploadPartFunc != nil {
		return m.UploadPartFunc(ctx, bucketName, objectKey, uploadID, partNumber, size, data)
	}
	return "\"test-etag\"", nil
}

func (m *MockBackend) CompleteMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string, partETags map[int]string) (string, error) {
	if m.CompleteMultipartUploadFunc != nil {
		return m.CompleteMultipartUploadFunc(ctx, bucketName, objectKey, uploadID, partETags)
	}
	return "\"combined-etag\"", nil
}

func (m *MockBackend) AbortMultipartUpload(ctx context.Context, bucketName, objectKey, uploadID string) error {
	if m.AbortMultipartUploadFunc != nil {
		return m.AbortMultipartUploadFunc(ctx, bucketName, objectKey, uploadID)
	}
	return nil
}

func (m *MockBackend) ListParts(ctx context.Context, bucketName, objectKey, uploadID string) ([]interface{}, error) {
	if m.ListPartsFunc != nil {
		return m.ListPartsFunc(ctx, bucketName, objectKey, uploadID)
	}
	return []interface{}{
		map[string]interface{}{
			"PartNumber": 1,
			"ETag":       "\"part1-etag\"",
			"Size":       int64(1024),
		},
	}, nil
}

func (m *MockBackend) ListMultipartUploads(ctx context.Context, bucketName string) ([]interface{}, error) {
	if m.ListMultipartUploadsFunc != nil {
		return m.ListMultipartUploadsFunc(ctx, bucketName)
	}
	return []interface{}{}, nil
}

func (m *MockBackend) EnableVersioning(ctx context.Context, bucketName string) error {
	if m.EnableVersioningFunc != nil {
		return m.EnableVersioningFunc(ctx, bucketName)
	}
	return nil
}

func (m *MockBackend) GetVersioning(ctx context.Context, bucketName string) (bool, error) {
	if m.GetVersioningFunc != nil {
		return m.GetVersioningFunc(ctx, bucketName)
	}
	return false, nil
}

func (m *MockBackend) ListObjectVersions(ctx context.Context, bucketName, prefix string) ([]interface{}, error) {
	if m.ListObjectVersionsFunc != nil {
		return m.ListObjectVersionsFunc(ctx, bucketName, prefix)
	}
	return []interface{}{}, nil
}

func (m *MockBackend) GetObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) (backend.ObjectInfo, error) {
	if m.GetObjectVersionFunc != nil {
		return m.GetObjectVersionFunc(ctx, bucketName, objectKey, versionID)
	}
	return backend.ObjectInfo{Body: io.NopCloser(bytes.NewReader([]byte("test data"))), LastModified: time.Now()}, nil
}

func (m *MockBackend) DeleteObjectVersion(ctx context.Context, bucketName, objectKey, versionID string) error {
	if m.DeleteObjectVersionFunc != nil {
		return m.DeleteObjectVersionFunc(ctx, bucketName, objectKey, versionID)
	}
	return nil
}

func TestS3HandlerCreation(t *testing.T) {
	mockBackend := &MockBackend{}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
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
			defer func() { _ = logger.Sync() }()
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

func TestHeadBucketHandler(t *testing.T) {
	tests := []struct {
		name           string
		bucketExists   bool
		headBucketErr  error
		expectedStatus int
	}{
		{
			name:           "bucket exists",
			bucketExists:   true,
			headBucketErr:  nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "bucket does not exist",
			bucketExists:   false,
			headBucketErr:  nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "backend error",
			bucketExists:   false,
			headBucketErr:  errors.New("backend error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackend := &MockBackend{
				HeadBucketFunc: func(ctx context.Context, bucketName string) (bool, error) {
					return tt.bucketExists, tt.headBucketErr
				},
			}
			logger, _ := zap.NewDevelopment()
			defer func() { _ = logger.Sync() }()
			handler := NewS3Handler(mockBackend, logger)

			r := chi.NewRouter()
			r.Head("/{bucket}", handler.HeadBucketHandler)

			req := httptest.NewRequest("HEAD", "/test-bucket", nil)
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
			defer func() { _ = logger.Sync() }()
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
			defer func() { _ = logger.Sync() }()
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
		ListObjectsV2Func: func(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) ([]string, string, error) {
			if continuationToken != "" {
				t.Fatalf("unexpected continuation token: %s", continuationToken)
			}
			if maxResults != 1000 {
				t.Fatalf("expected default maxResults 1000, got %d", maxResults)
			}
			return []string{"obj1", "obj2"}, "", nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
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

func TestListObjectsV2Handler_WithContinuation(t *testing.T) {
	mockBackend := &MockBackend{
		ListObjectsV2Func: func(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) ([]string, string, error) {
			if continuationToken != "start-token" {
				t.Fatalf("expected continuation token start-token, got %s", continuationToken)
			}
			if maxResults != 2 {
				t.Fatalf("expected maxResults 2, got %d", maxResults)
			}
			return []string{"a", "b"}, "next-token", nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}", handler.ListObjectsV2Handler)

	req := httptest.NewRequest("GET", "/mybucket?list-type=2&prefix=p/&continuation-token=start-token&max-keys=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp models.ListObjectsV2Response
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.NextContinuationToken != "next-token" {
		t.Fatalf("expected next continuation token next-token, got %s", resp.NextContinuationToken)
	}
	if !resp.IsTruncated {
		t.Fatalf("expected response to be truncated")
	}
	if resp.KeyCount != 2 {
		t.Fatalf("expected key count 2, got %d", resp.KeyCount)
	}
}

func TestListObjectsV2Handler_MaxKeysClamp(t *testing.T) {
	mockBackend := &MockBackend{
		ListObjectsV2Func: func(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) ([]string, string, error) {
			if continuationToken != "tok" {
				t.Fatalf("expected continuation token tok, got %s", continuationToken)
			}
			if maxResults != 1000 { // expect clamp to 1000 when client asks for too many
				t.Fatalf("expected maxResults 1000, got %d", maxResults)
			}
			return []string{"only"}, "next", nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}", handler.ListObjectsV2Handler)

	req := httptest.NewRequest("GET", "/mybucket?list-type=2&continuation-token=tok&max-keys=5001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp models.ListObjectsV2Response
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ContinuationToken != "tok" {
		t.Fatalf("expected continuation token echoed, got %s", resp.ContinuationToken)
	}
	if resp.NextContinuationToken != "next" {
		t.Fatalf("expected next continuation token next, got %s", resp.NextContinuationToken)
	}
	if resp.KeyCount != 1 {
		t.Fatalf("expected key count 1, got %d", resp.KeyCount)
	}
	if !resp.IsTruncated {
		t.Fatalf("expected response truncated")
	}
}

func TestListObjectsV2Handler_InvalidMaxKeys(t *testing.T) {
	mockBackend := &MockBackend{
		ListObjectsV2Func: func(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) ([]string, string, error) {
			if maxResults != 1000 {
				t.Fatalf("expected default maxResults 1000 on parse failure, got %d", maxResults)
			}
			return []string{"one"}, "", nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}", handler.ListObjectsV2Handler)

	req := httptest.NewRequest("GET", "/mybucket?list-type=2&max-keys=not-a-number", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp models.ListObjectsV2Response
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.KeyCount != 1 {
		t.Fatalf("expected key count 1, got %d", resp.KeyCount)
	}
	if resp.IsTruncated {
		t.Fatalf("expected not truncated")
	}
}

func TestPutObjectHandler(t *testing.T) {
	mockBackend := &MockBackend{
		PutObjectFunc: func(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) error {
			return nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
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
		GetObjectFunc: func(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error) {
			return backend.ObjectInfo{Body: io.NopCloser(bytes.NewReader([]byte("test data"))), LastModified: time.Now()}, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
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
		HeadObjectFunc: func(ctx context.Context, bucketName, objectKey string) (bool, int64, time.Time, error) {
			return true, int64(9), time.Now(), nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
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
	defer func() { _ = logger.Sync() }()
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

func TestInitiateMultipartUploadHandler(t *testing.T) {
	mockBackend := &MockBackend{
		InitiateMultipartUploadFunc: func(ctx context.Context, bucketName, objectKey string) (string, error) {
			return "upload-123", nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Post("/{bucket}/*", handler.InitiateMultipartUploadHandler)

	req := httptest.NewRequest("POST", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("InitiateMultipartUpload: expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("upload-123")) {
		t.Errorf("InitiateMultipartUpload: expected upload ID in response, got %s", body)
	}
}

func TestUploadPartHandler(t *testing.T) {
	tests := []struct {
		name       string
		uploadID   string
		partNumber string
		wantStatus int
		wantError  bool
	}{
		{name: "valid part upload", uploadID: "upload-123", partNumber: "1", wantStatus: http.StatusOK, wantError: false},
		{name: "missing uploadId", uploadID: "", partNumber: "1", wantStatus: http.StatusInternalServerError, wantError: false},
		{name: "missing partNumber", uploadID: "upload-123", partNumber: "", wantStatus: http.StatusInternalServerError, wantError: false},
		{name: "invalid partNumber", uploadID: "upload-123", partNumber: "abc", wantStatus: http.StatusInternalServerError, wantError: false},
		{name: "partNumber out of range", uploadID: "upload-123", partNumber: "99999", wantStatus: http.StatusInternalServerError, wantError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackend := &MockBackend{
				UploadPartFunc: func(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, size int64, data io.Reader) (string, error) {
					if tt.wantError {
						return "", errors.New("upload failed")
					}
					return "\"part-etag\"", nil
				},
			}
			logger, _ := zap.NewDevelopment()
			defer func() { _ = logger.Sync() }()
			handler := NewS3Handler(mockBackend, logger)

			r := chi.NewRouter()
			r.Put("/{bucket}/{key}", handler.UploadPartHandler)

			path := "/mybucket/mykey"
			if tt.uploadID != "" {
				path += "?uploadId=" + tt.uploadID
			}
			if tt.partNumber != "" {
				if tt.uploadID != "" {
					path += "&partNumber=" + tt.partNumber
				} else {
					path += "?partNumber=" + tt.partNumber
				}
			}

			req := httptest.NewRequest("PUT", path, bytes.NewReader([]byte("part data")))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestCompleteMultipartUploadHandler(t *testing.T) {
	mockBackend := &MockBackend{
		CompleteMultipartUploadFunc: func(ctx context.Context, bucketName, objectKey, uploadID string, partETags map[int]string) (string, error) {
			return "\"combined-etag\"", nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Post("/{bucket}/*", handler.CompleteMultipartUploadHandler)

	requestBody := `<?xml version="1.0" encoding="UTF-8"?>
<CompleteMultipartUpload>
  <Part>
    <PartNumber>1</PartNumber>
    <ETag>"part-etag-1"</ETag>
  </Part>
</CompleteMultipartUpload>`

	req := httptest.NewRequest("POST", "/mybucket/mykey?uploadId=upload-123", bytes.NewReader([]byte(requestBody)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CompleteMultipartUpload: expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("combined-etag")) {
		t.Errorf("CompleteMultipartUpload: expected combined etag in response")
	}
}

func TestCompleteMultipartUploadHandler_MissingUploadID(t *testing.T) {
	mockBackend := &MockBackend{}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Post("/{bucket}/*", handler.CompleteMultipartUploadHandler)

	requestBody := `<?xml version="1.0" encoding="UTF-8"?>
<CompleteMultipartUpload>
  <Part>
    <PartNumber>1</PartNumber>
    <ETag>"part-etag-1"</ETag>
  </Part>
</CompleteMultipartUpload>`

	req := httptest.NewRequest("POST", "/mybucket/mykey", bytes.NewReader([]byte(requestBody)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAbortMultipartUploadHandler(t *testing.T) {
	mockBackend := &MockBackend{
		AbortMultipartUploadFunc: func(ctx context.Context, bucketName, objectKey, uploadID string) error {
			return nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Delete("/{bucket}/*", handler.AbortMultipartUploadHandler)

	req := httptest.NewRequest("DELETE", "/mybucket/mykey?uploadId=upload-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("AbortMultipartUpload: expected status 204, got %d", w.Code)
	}
}

func TestAbortMultipartUploadHandler_MissingUploadID(t *testing.T) {
	mockBackend := &MockBackend{}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Delete("/{bucket}/*", handler.AbortMultipartUploadHandler)

	req := httptest.NewRequest("DELETE", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestListPartsHandler(t *testing.T) {
	mockBackend := &MockBackend{
		ListPartsFunc: func(ctx context.Context, bucketName, objectKey, uploadID string) ([]interface{}, error) {
			return []interface{}{
				map[string]interface{}{
					"PartNumber": 1,
					"ETag":       "\"part1-etag\"",
					"Size":       int64(1024),
				},
				map[string]interface{}{
					"PartNumber": 2,
					"ETag":       "\"part2-etag\"",
					"Size":       int64(1024),
				},
			}, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}/*", handler.ListPartsHandler)

	req := httptest.NewRequest("GET", "/mybucket/mykey?uploadId=upload-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListParts: expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("PartNumber")) {
		t.Errorf("ListParts: expected parts in response")
	}
}

func TestListMultipartUploadsHandler(t *testing.T) {
	mockBackend := &MockBackend{
		ListMultipartUploadsFunc: func(ctx context.Context, bucketName string) ([]interface{}, error) {
			return []interface{}{
				map[string]interface{}{
					"Key":       "testkey1",
					"UploadID":  "upload-123",
					"Initiated": "2025-01-01T00:00:00Z",
				},
			}, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}", handler.ListMultipartUploadsHandler)

	req := httptest.NewRequest("GET", "/mybucket?uploads", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListMultipartUploads: expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("testkey1")) {
		t.Errorf("ListMultipartUploads: expected upload info in response")
	}
}

// TestEnableVersioningHandler tests enabling versioning on a bucket
func TestEnableVersioningHandler(t *testing.T) {
	tests := []struct {
		name                 string
		bucket               string
		enableVersioningFunc func(ctx context.Context, bucketName string) error
		expectStatus         int
	}{
		{
			name:   "enable_versioning_success",
			bucket: "mybucket",
			enableVersioningFunc: func(ctx context.Context, bucketName string) error {
				return nil
			},
			expectStatus: http.StatusOK,
		},
		{
			name:   "enable_versioning_error",
			bucket: "mybucket",
			enableVersioningFunc: func(ctx context.Context, bucketName string) error {
				return errors.New("containernotfound")
			},
			expectStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockBackend := &MockBackend{
				EnableVersioningFunc: test.enableVersioningFunc,
			}
			logger, _ := zap.NewDevelopment()
			defer func() { _ = logger.Sync() }()
			handler := NewS3Handler(mockBackend, logger)

			r := chi.NewRouter()
			r.Put("/{bucket}", handler.EnableVersioningHandler)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("PUT", "/"+test.bucket+"?versioning", nil)
			r.ServeHTTP(w, req)

			if w.Code != test.expectStatus {
				t.Errorf("EnableVersioning: expected status %d, got %d", test.expectStatus, w.Code)
			}
		})
	}
}

// TestGetVersioningHandler tests getting versioning status
func TestGetVersioningHandler(t *testing.T) {
	tests := []struct {
		name              string
		bucket            string
		getVersioningFunc func(ctx context.Context, bucketName string) (bool, error)
		expectStatus      int
	}{
		{
			name:   "get_versioning_enabled",
			bucket: "mybucket",
			getVersioningFunc: func(ctx context.Context, bucketName string) (bool, error) {
				return true, nil
			},
			expectStatus: http.StatusOK,
		},
		{
			name:   "get_versioning_disabled",
			bucket: "mybucket",
			getVersioningFunc: func(ctx context.Context, bucketName string) (bool, error) {
				return false, nil
			},
			expectStatus: http.StatusOK,
		},
		{
			name:   "get_versioning_error",
			bucket: "mybucket",
			getVersioningFunc: func(ctx context.Context, bucketName string) (bool, error) {
				return false, errors.New("containernotfound")
			},
			expectStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockBackend := &MockBackend{
				GetVersioningFunc: test.getVersioningFunc,
			}
			logger, _ := zap.NewDevelopment()
			defer func() { _ = logger.Sync() }()
			handler := NewS3Handler(mockBackend, logger)

			r := chi.NewRouter()
			r.Get("/{bucket}", handler.GetVersioningHandler)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/"+test.bucket+"?versioning", nil)
			r.ServeHTTP(w, req)

			if w.Code != test.expectStatus {
				t.Errorf("GetVersioning: expected status %d, got %d", test.expectStatus, w.Code)
			}
		})
	}
}

// TestListObjectVersionsHandler tests listing object versions
func TestListObjectVersionsHandler(t *testing.T) {
	mockBackend := &MockBackend{
		ListObjectVersionsFunc: func(ctx context.Context, bucketName, prefix string) ([]interface{}, error) {
			return []interface{}{}, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}", handler.ListObjectVersionsHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/mybucket?versions", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListObjectVersions: expected status 200, got %d", w.Code)
	}
}

// TestGetObjectVersionHandler tests retrieving a specific object version
func TestGetObjectVersionHandler(t *testing.T) {
	mockBackend := &MockBackend{
		GetObjectVersionFunc: func(ctx context.Context, bucketName, objectKey, versionID string) (backend.ObjectInfo, error) {
			return backend.ObjectInfo{Body: io.NopCloser(bytes.NewReader([]byte("version data"))), LastModified: time.Now()}, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}/*", handler.GetObjectVersionHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/mybucket/mykey?versionId=v123", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetObjectVersion: expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body != "version data" {
		t.Errorf("GetObjectVersion: expected body 'version data', got %s", body)
	}
}

// TestDeleteObjectVersionHandler tests deleting a specific object version
func TestDeleteObjectVersionHandler(t *testing.T) {
	tests := []struct {
		name         string
		deleteFunc   func(ctx context.Context, bucketName, objectKey, versionID string) error
		expectStatus int
	}{
		{
			name: "delete_version_success",
			deleteFunc: func(ctx context.Context, bucketName, objectKey, versionID string) error {
				return nil
			},
			expectStatus: http.StatusNoContent,
		},
		{
			name: "delete_version_error",
			deleteFunc: func(ctx context.Context, bucketName, objectKey, versionID string) error {
				return errors.New("blobnotfound")
			},
			expectStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockBackend := &MockBackend{
				DeleteObjectVersionFunc: test.deleteFunc,
			}
			logger, _ := zap.NewDevelopment()
			defer func() { _ = logger.Sync() }()
			handler := NewS3Handler(mockBackend, logger)

			r := chi.NewRouter()
			r.Delete("/{bucket}/*", handler.DeleteObjectVersionHandler)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("DELETE", "/mybucket/mykey?versionId=v123", nil)
			r.ServeHTTP(w, req)

			if w.Code != test.expectStatus {
				t.Errorf("DeleteObjectVersion: expected status %d, got %d", test.expectStatus, w.Code)
			}
		})
	}
}

// TestEnableVersioningHandler_Error tests error handling in EnableVersioning
func TestEnableVersioningHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		EnableVersioningFunc: func(ctx context.Context, bucketName string) error {
			return errors.New("bucket not found")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Put("/{bucket}", handler.EnableVersioningHandler)

	body := bytes.NewReader([]byte("<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>"))
	req := httptest.NewRequest("PUT", "/mybucket?versioning", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestGetVersioningHandler_Error tests error handling in GetVersioning
func TestGetVersioningHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		GetVersioningFunc: func(ctx context.Context, bucketName string) (bool, error) {
			return false, errors.New("service error")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}", handler.GetVersioningHandler)

	req := httptest.NewRequest("GET", "/mybucket?versioning", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestListObjectVersionsHandler_Error tests error handling in ListObjectVersions
func TestListObjectVersionsHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		ListObjectVersionsFunc: func(ctx context.Context, bucketName, prefix string) ([]interface{}, error) {
			return nil, errors.New("list failed")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}/*", handler.ListObjectVersionsHandler)

	req := httptest.NewRequest("GET", "/mybucket/?list-type=2&versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestGetObjectVersionHandler_Error tests error handling in GetObjectVersion
func TestGetObjectVersionHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		GetObjectVersionFunc: func(ctx context.Context, bucketName, objectKey, versionID string) (backend.ObjectInfo, error) {
			return backend.ObjectInfo{}, errors.New("version not found")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}/*", handler.GetObjectVersionHandler)

	req := httptest.NewRequest("GET", "/mybucket/mykey?versionId=v123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestListPartsHandler_Error tests error handling in ListParts
func TestListPartsHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		ListPartsFunc: func(ctx context.Context, bucketName, objectKey, uploadID string) ([]interface{}, error) {
			return nil, errors.New("upload not found")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}/*", handler.ListPartsHandler)

	req := httptest.NewRequest("GET", "/mybucket/mykey?uploadId=123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestListMultipartUploadsHandler_Error tests error handling in ListMultipartUploads
func TestListMultipartUploadsHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		ListMultipartUploadsFunc: func(ctx context.Context, bucketName string) ([]interface{}, error) {
			return nil, errors.New("service error")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}", handler.ListMultipartUploadsHandler)

	req := httptest.NewRequest("GET", "/mybucket?uploads", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestListBucketsHandler_Error tests error handling in ListBuckets
func TestListBucketsHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		ListBucketsFunc: func(ctx context.Context) ([]string, error) {
			return nil, errors.New("service unavailable")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/", handler.ListBucketsHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestCreateBucketHandler_Error tests error handling in CreateBucket
func TestCreateBucketHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		CreateBucketFunc: func(ctx context.Context, bucketName string) error {
			return errors.New("service error")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Put("/{bucket}", handler.CreateBucketHandler)

	req := httptest.NewRequest("PUT", "/mybucket", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestDeleteBucketHandler_Error tests error handling in DeleteBucket
func TestDeleteBucketHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		DeleteBucketFunc: func(ctx context.Context, bucketName string) error {
			return errors.New("service error")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Delete("/{bucket}", handler.DeleteBucketHandler)

	req := httptest.NewRequest("DELETE", "/mybucket", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestGetObjectHandler_Error tests error handling in GetObject
func TestGetObjectHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		GetObjectFunc: func(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error) {
			return backend.ObjectInfo{}, errors.New("not found")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}/*", handler.GetObjectHandler)

	req := httptest.NewRequest("GET", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestPutObjectHandler_Error tests error handling in PutObject
func TestPutObjectHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		PutObjectFunc: func(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) error {
			return errors.New("upload failed")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Put("/{bucket}/*", handler.PutObjectHandler)

	body := bytes.NewReader([]byte("test data"))
	req := httptest.NewRequest("PUT", "/mybucket/mykey", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestDeleteObjectHandler_Error tests error handling in DeleteObject
func TestDeleteObjectHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		DeleteObjectFunc: func(ctx context.Context, bucketName, objectKey string) error {
			return errors.New("delete failed")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Delete("/{bucket}/*", handler.DeleteObjectHandler)

	req := httptest.NewRequest("DELETE", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestHeadObjectHandler_NotFound tests 404 behavior in HeadObject
func TestHeadObjectHandler_NotFound(t *testing.T) {
	mockBackend := &MockBackend{
		HeadObjectFunc: func(ctx context.Context, bucketName, objectKey string) (bool, int64, time.Time, error) {
			return false, 0, time.Time{}, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Head("/{bucket}/*", handler.HeadObjectHandler)

	req := httptest.NewRequest("HEAD", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestListObjectsV2Handler_Error tests error handling in ListObjects
func TestListObjectsV2Handler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		ListObjectsV2Func: func(ctx context.Context, bucketName, prefix, continuationToken string, maxResults int32) ([]string, string, error) {
			return nil, "", errors.New("list failed")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}/*", handler.ListObjectsV2Handler)

	req := httptest.NewRequest("GET", "/mybucket/?list-type=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestInitiateMultipartUploadHandler_Error tests error handling in InitiateMultipartUpload
func TestInitiateMultipartUploadHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		InitiateMultipartUploadFunc: func(ctx context.Context, bucketName, objectKey string) (string, error) {
			return "", errors.New("initiate failed")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Post("/{bucket}/*", handler.InitiateMultipartUploadHandler)

	req := httptest.NewRequest("POST", "/mybucket/mykey?uploads", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestUploadPartHandler_Error tests error handling in UploadPart
func TestUploadPartHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		UploadPartFunc: func(ctx context.Context, bucketName, objectKey, uploadID string, partNumber int, size int64, data io.Reader) (string, error) {
			return "", errors.New("upload part failed")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Put("/{bucket}/*", handler.UploadPartHandler)

	body := bytes.NewReader([]byte("part data"))
	req := httptest.NewRequest("PUT", "/mybucket/mykey?uploadId=123&partNumber=1", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestCompleteMultipartUploadHandler_Error tests error handling in CompleteMultipartUpload
func TestCompleteMultipartUploadHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		CompleteMultipartUploadFunc: func(ctx context.Context, bucketName, objectKey, uploadID string, partETags map[int]string) (string, error) {
			return "", errors.New("complete failed")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Post("/{bucket}/*", handler.CompleteMultipartUploadHandler)

	body := bytes.NewReader([]byte("<CompleteMultipartUpload></CompleteMultipartUpload>"))
	req := httptest.NewRequest("POST", "/mybucket/mykey?uploadId=123", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestAbortMultipartUploadHandler_Error tests error handling in AbortMultipartUpload
func TestAbortMultipartUploadHandler_Error(t *testing.T) {
	mockBackend := &MockBackend{
		AbortMultipartUploadFunc: func(ctx context.Context, bucketName, objectKey, uploadID string) error {
			return errors.New("abort failed")
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Delete("/{bucket}/*", handler.AbortMultipartUploadHandler)

	req := httptest.NewRequest("DELETE", "/mybucket/mykey?uploadId=123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestS3HandlerStatsIntegration tests stats recording in handlers
func TestS3HandlerStatsIntegration(t *testing.T) {
	mockBackend := &MockBackend{
		ListBucketsFunc: func(ctx context.Context) ([]string, error) {
			return []string{"bucket1", "bucket2"}, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	// Verify stats are initialized
	stats := handler.GetStats()
	if stats.TotalRequests != 0 {
		t.Errorf("initial stats should have 0 requests, got %d", stats.TotalRequests)
	}

	r := chi.NewRouter()
	r.Get("/", handler.ListBucketsHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify stats were updated
	stats = handler.GetStats()
	if stats.TotalRequests != 1 {
		t.Errorf("after one request, stats should show 1 request, got %d", stats.TotalRequests)
	}
}

// TestPostObjectHandler tests the default POST handler (if implemented)
func TestPostObjectHandler(t *testing.T) {
	mockBackend := &MockBackend{
		ListObjectsFunc: func(ctx context.Context, bucketName, prefix string) ([]string, error) {
			return []string{"key1", "key2"}, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Post("/{bucket}/*", handler.PostObjectHandler)

	body := bytes.NewReader([]byte("test data"))
	req := httptest.NewRequest("POST", "/mybucket/mykey", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// PostObjectHandler may not be fully implemented, so we just verify it doesn't crash
	if w.Code == 0 {
		t.Errorf("PostObjectHandler should set a status code")
	}
}

// TestDeleteObjectHandler_Success tests successful object deletion
func TestDeleteObjectHandler_Success(t *testing.T) {
	mockBackend := &MockBackend{
		DeleteObjectFunc: func(ctx context.Context, bucketName, objectKey string) error {
			return nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Delete("/{bucket}/*", handler.DeleteObjectHandler)

	req := httptest.NewRequest("DELETE", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

// TestHeadObjectHandler_Success tests successful object existence check
func TestHeadObjectHandler_Success(t *testing.T) {
	mockBackend := &MockBackend{
		HeadObjectFunc: func(ctx context.Context, bucketName, objectKey string) (bool, int64, time.Time, error) {
			return true, int64(9), time.Now(), nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Head("/{bucket}/*", handler.HeadObjectHandler)

	req := httptest.NewRequest("HEAD", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestGetObjectHandler_Success tests successful object retrieval
func TestGetObjectHandler_Success(t *testing.T) {
	mockBackend := &MockBackend{
		GetObjectFunc: func(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error) {
			return backend.ObjectInfo{Body: io.NopCloser(bytes.NewReader([]byte("test object data"))), LastModified: time.Now()}, nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Get("/{bucket}/*", handler.GetObjectHandler)

	req := httptest.NewRequest("GET", "/mybucket/mykey", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "test object data" {
		t.Errorf("expected body 'test object data', got %s", w.Body.String())
	}
}

// TestPutObjectHandler_Success tests successful object upload
func TestPutObjectHandler_Success(t *testing.T) {
	mockBackend := &MockBackend{
		PutObjectFunc: func(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) error {
			return nil
		},
	}
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	handler := NewS3Handler(mockBackend, logger)

	r := chi.NewRouter()
	r.Put("/{bucket}/*", handler.PutObjectHandler)

	body := bytes.NewReader([]byte("upload data"))
	req := httptest.NewRequest("PUT", "/mybucket/mykey", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestSetCacheManager tests setting cache manager on handler
func TestSetCacheManager(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	backend := &MockBackend{}
	handler := NewS3Handler(backend, logger)

	// Mock cache manager
	tmpDir := t.TempDir()
	cm, err := cache.NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	handler.SetCacheManager(cm)
	if handler.cacheManager != cm {
		t.Fatal("SetCacheManager did not set the cache manager")
	}
}

// TestGenerateCacheKey tests cache key generation
func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		bucket   string
		key      string
		expected string
	}{
		{"mybucket", "mykey", "mybucket/mykey"},
		{"bucket1", "path/to/object", "bucket1/path/to/object"},
		{"test-bucket", "test-key", "test-bucket/test-key"},
		{"bucket", "", "bucket/"},
		{"", "key", "/key"},
	}

	for _, tt := range tests {
		result := generateCacheKey(tt.bucket, tt.key)
		if result != tt.expected {
			t.Errorf("generateCacheKey(%q, %q) = %q, expected %q", tt.bucket, tt.key, result, tt.expected)
		}
	}
}

// TestGetObjectHandler_WithCache tests GetObject with cache enabled
func TestGetObjectHandler_WithCache(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	tmpDir := t.TempDir()
	cm, err := cache.NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	objectData := []byte("cached object data")
	backend := &MockBackend{
		GetObjectFunc: func(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error) {
			return backend.ObjectInfo{Body: io.NopCloser(bytes.NewReader(objectData)), LastModified: time.Now()}, nil
		},
	}

	handler := NewS3Handler(backend, logger)
	handler.SetCacheManager(cm)

	r := chi.NewRouter()
	r.Get("/{bucket}/*", handler.GetObjectHandler)

	req := httptest.NewRequest("GET", "/bucket1/key1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != string(objectData) {
		t.Errorf("expected body %s, got %s", objectData, w.Body.String())
	}
}

// TestGetObjectHandler_CacheHit tests GetObject with cache hit
func TestGetObjectHandler_CacheHit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	tmpDir := t.TempDir()
	cm, err := cache.NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	objectData := []byte("cached object data for hit test")
	cacheKey := "bucket1/key1"

	// Pre-populate cache
	err = cm.CacheObject(cacheKey, objectData)
	if err != nil {
		t.Fatalf("CacheObject failed: %v", err)
	}

	callCount := 0
	backend := &MockBackend{
		GetObjectFunc: func(ctx context.Context, bucketName, objectKey string) (backend.ObjectInfo, error) {
			callCount++
			return backend.ObjectInfo{Body: io.NopCloser(bytes.NewReader(objectData)), LastModified: time.Now()}, nil
		},
	}

	handler := NewS3Handler(backend, logger)
	handler.SetCacheManager(cm)

	r := chi.NewRouter()
	r.Get("/{bucket}/*", handler.GetObjectHandler)

	req := httptest.NewRequest("GET", "/bucket1/key1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that the cached version was served (should not have called backend)
	if w.Header().Get("X-Cache-Hit") != "true" {
		t.Errorf("expected X-Cache-Hit header to be true")
	}

	if callCount != 0 {
		t.Errorf("backend should not have been called for cache hit, but was called %d times", callCount)
	}
}

// TestHeadObjectHandler_WithCache tests HeadObject with cache
func TestHeadObjectHandler_WithCache(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	tmpDir := t.TempDir()
	cm, err := cache.NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	backend := &MockBackend{
		HeadObjectFunc: func(ctx context.Context, bucketName, objectKey string) (bool, int64, time.Time, error) {
			return true, int64(9), time.Now(), nil
		},
	}

	handler := NewS3Handler(backend, logger)
	handler.SetCacheManager(cm)

	r := chi.NewRouter()
	r.Head("/{bucket}/*", handler.HeadObjectHandler)

	req := httptest.NewRequest("HEAD", "/bucket1/key1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestPutObjectHandler_WithCache tests PutObject with cache enabled
func TestPutObjectHandler_WithCache(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()
	tmpDir := t.TempDir()
	cm, err := cache.NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	backend := &MockBackend{
		PutObjectFunc: func(ctx context.Context, bucketName, objectKey string, size int64, data io.Reader) error {
			return nil
		},
	}

	handler := NewS3Handler(backend, logger)
	handler.SetCacheManager(cm)

	r := chi.NewRouter()
	r.Put("/{bucket}/*", handler.PutObjectHandler)

	body := bytes.NewReader([]byte("test data"))
	req := httptest.NewRequest("PUT", "/bucket1/key1", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestPutObjectHandler_CopyObject(t *testing.T) {
	logger := zap.NewNop()
	mockBackend := &MockBackend{}
	handler := NewS3Handler(mockBackend, logger)

	t.Run("successful copy", func(t *testing.T) {
		mockBackend.CopyObjectFunc = func(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
			if srcBucket != "source-bucket" || srcKey != "source-key" {
				return errors.New("unexpected source")
			}
			if destBucket != "dest-bucket" || destKey != "dest-key" {
				return errors.New("unexpected destination")
			}
			return nil
		}

		req := httptest.NewRequest("PUT", "/dest-bucket/dest-key", nil)
		req.Header.Set("x-amz-copy-source", "/source-bucket/source-key")

		// Setup chi context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("bucket", "dest-bucket")
		rctx.URLParams.Add("*", "dest-key")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.PutObjectHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		// Check response body for CopyObjectResult
		if !strings.Contains(w.Body.String(), "CopyObjectResult") {
			t.Errorf("expected CopyObjectResult in body, got %s", w.Body.String())
		}
	})

	t.Run("invalid copy source", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/dest-bucket/dest-key", nil)
		req.Header.Set("x-amz-copy-source", "invalid-format")

		// Setup chi context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("bucket", "dest-bucket")
		rctx.URLParams.Add("*", "dest-key")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.PutObjectHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestDeleteObjectsHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	backend := &MockBackend{
		DeleteObjectFunc: func(ctx context.Context, bucketName, objectKey string) error {
			if objectKey == "error" {
				return errors.New("delete failed")
			}
			return nil
		},
	}
	handler := NewS3Handler(backend, logger)

	t.Run("success", func(t *testing.T) {
		body := `
		<Delete>
			<Object>
				<Key>obj1</Key>
			</Object>
			<Object>
				<Key>obj2</Key>
			</Object>
		</Delete>`
		req := httptest.NewRequest("POST", "/test-bucket?delete", strings.NewReader(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("bucket", "test-bucket")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.DeleteObjectsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("malformed xml", func(t *testing.T) {
		body := `<Delete><Object><Key>obj1</Key></Object>` // Missing closing Delete
		req := httptest.NewRequest("POST", "/test-bucket?delete", strings.NewReader(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("bucket", "test-bucket")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		handler.DeleteObjectsHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestHeadObjectHandler_CacheHit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	tmpDir := t.TempDir()
	cm, err := cache.NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	// Pre-populate cache
	content := []byte("cached content")
	err = cm.CacheObject("bucket1/key1", content)
	if err != nil {
		t.Fatalf("CacheObject failed: %v", err)
	}

	backend := &MockBackend{
		HeadObjectFunc: func(ctx context.Context, bucketName, objectKey string) (bool, int64, time.Time, error) {
			t.Error("Backend HeadObject should not be called on cache hit")
			return true, int64(14), time.Now(), nil
		},
	}

	handler := NewS3Handler(backend, logger)
	handler.SetCacheManager(cm)

	r := chi.NewRouter()
	r.Head("/{bucket}/*", handler.HeadObjectHandler)

	req := httptest.NewRequest("HEAD", "/bucket1/key1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("X-Cache-Hit") != "true" {
		t.Error("expected X-Cache-Hit header to be true")
	}

	if w.Header().Get("Content-Length") != "14" { // len("cached content")
		t.Errorf("expected Content-Length 14, got %s", w.Header().Get("Content-Length"))
	}
}
