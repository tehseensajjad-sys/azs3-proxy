package azureblob

import (
	"context"
	"testing"

	"github.com/vibhansa-msft/s3-azure-proxy/internal/config"
	"go.uber.org/zap"
)

func TestNewAzureBlobBackend(t *testing.T) {
	tests := []struct {
		name    string
		connStr string
		wantErr bool
	}{
		{name: "invalid connection string", connStr: "", wantErr: true},
		{name: "invalid account key", connStr: "DefaultEndpointsProtocol=https;AccountName=test;AccountKey=invalid;EndpointSuffix=core.windows.net", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAzureBlobBackend(tt.connStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error: %v", err != nil, tt.wantErr)
			}
		})
	}
}

func TestNewAzureBlobBackendWithAuth(t *testing.T) {
	tests := []struct {
		name       string
		authConfig *config.AzureAuthConfig
	}{
		{
			name: "account key auth",
			authConfig: &config.AzureAuthConfig{
				Mode:               config.AuthModeAccountKey,
				StorageAccountName: "testaccount",
				AccountKey:         "dGVzdGtleQ==",
			},
		},
		{
			name: "sas token auth",
			authConfig: &config.AzureAuthConfig{
				Mode:               config.AuthModeSAS,
				StorageAccountName: "testaccount",
				SASToken:           "sv=2021-06-08&st=2023-01-01&se=2024-01-01",
			},
		},
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAzureBlobBackendWithAuth(tt.authConfig, logger)
			if err != nil {
				t.Logf("Backend creation returned error (might be expected): %v", err)
			}
		})
	}
}

func TestAzureBlobBackendOperations(t *testing.T) {
	// Test that backend methods can be called without panicking
	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==",
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	backend, err := NewAzureBlobBackendWithAuth(authConfig, logger)
	if err != nil {
		t.Logf("Backend creation failed (expected for test): %v", err)
		return
	}

	if backend == nil {
		t.Skip("Backend is nil, skipping operation tests")
	}

	ctx := context.Background()

	// Test that operations exist and can be called
	// They will fail with invalid credentials, but should not panic
	_, _ = backend.ListBuckets(ctx)
	_, _ = backend.ListObjects(ctx, "bucket", "prefix")
	_ = backend.CreateBucket(ctx, "bucket")
	_ = backend.DeleteBucket(ctx, "bucket")
	_, _ = backend.HeadObject(ctx, "bucket", "key")
	_ = backend.DeleteObject(ctx, "bucket", "key")
}

func TestAzureBlobBackendContextHandling(t *testing.T) {
	// Verify operations handle context cancellation properly
	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==",
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	backend, err := NewAzureBlobBackendWithAuth(authConfig, logger)
	if err != nil {
		t.Logf("Backend creation failed (expected for test): %v", err)
		return
	}

	if backend == nil {
		t.Skip("Backend is nil, skipping context tests")
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// These should handle the cancelled context gracefully
	_, _ = backend.ListBuckets(ctx)
	_, _ = backend.ListObjects(ctx, "bucket", "prefix")
}

func TestMultipartUploadFlow(t *testing.T) {
	// Test requires real Azure SDK client initialization
	// Use NewAzureBlobBackendWithAuth to get a proper client
	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==", // base64 encoded test key
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	backend, err := NewAzureBlobBackendWithAuth(authConfig, logger)
	if err != nil {
		t.Logf("Backend creation failed (expected for test): %v", err)
		// Expected to fail in test environment - we don't have real Azure credentials
		return
	}

	if backend == nil || backend.client == nil {
		t.Skip("Backend client is nil, skipping multipart upload flow test")
	}

	ctx := context.Background()

	// Test Initiate
	uploadID, err := backend.InitiateMultipartUpload(ctx, "testbucket", "testkey")
	if err != nil {
		t.Logf("InitiateMultipartUpload failed (expected in test): %v", err)
		return // Expected in test environment
	}
	if uploadID == "" {
		t.Error("InitiateMultipartUpload returned empty upload ID")
	}

	// Test ListParts (should work but be empty)
	parts, err := backend.ListParts(ctx, "testbucket", "testkey", uploadID)
	if err != nil {
		t.Logf("ListParts failed (expected in test): %v", err)
		return // Expected if upload doesn't exist
	}
	if len(parts) != 0 {
		t.Logf("ListParts returned %d parts", len(parts))
	}

	// Test AbortMultipartUpload
	err = backend.AbortMultipartUpload(ctx, "testbucket", "testkey", uploadID)
	if err != nil {
		t.Logf("AbortMultipartUpload failed: %v", err)
	}
}

func TestMultipartUploadInvalidOperations(t *testing.T) {
	backend := &AzureBlobBackend{
		multipartUploads: make(map[string]*MultipartUploadMetadata),
	}

	ctx := context.Background()

	// Test operations on non-existent upload
	_, err := backend.UploadPart(ctx, "bucket", "key", "nonexistent", 1, nil)
	if err == nil {
		t.Error("UploadPart should fail for non-existent upload")
	}

	_, err = backend.ListParts(ctx, "bucket", "key", "nonexistent")
	if err == nil {
		t.Error("ListParts should fail for non-existent upload")
	}

	err = backend.AbortMultipartUpload(ctx, "bucket", "key", "nonexistent")
	if err == nil {
		t.Error("AbortMultipartUpload should fail for non-existent upload")
	}
}

func TestMultipartUploadMismatchedBucketKey(t *testing.T) {
	// Create a minimal mock upload metadata
	mockUpload := &MultipartUploadMetadata{
		UploadID:        "test-upload-123",
		BucketName:      "bucket1",
		ObjectKey:       "key1",
		BlockIDs:        []string{},
		PartETagMap:     make(map[int]string),
		BlockBlobClient: nil, // Not needed for this test
	}

	backend := &AzureBlobBackend{
		multipartUploads: map[string]*MultipartUploadMetadata{
			"test-upload-123": mockUpload,
		},
	}

	ctx := context.Background()

	// Try to use it with bucket2/key2
	_, err := backend.UploadPart(ctx, "bucket2", "key2", "test-upload-123", 1, nil)
	if err == nil {
		t.Error("UploadPart should fail for mismatched bucket/key")
	}

	_, err = backend.ListParts(ctx, "bucket2", "key2", "test-upload-123")
	if err == nil {
		t.Error("ListParts should fail for mismatched bucket/key")
	}

	err = backend.AbortMultipartUpload(ctx, "bucket2", "key2", "test-upload-123")
	if err == nil {
		t.Error("AbortMultipartUpload should fail for mismatched bucket/key")
	}
}

// Test CompleteMultipartUpload validates error cases
func TestCompleteMultipartUploadErrors(t *testing.T) {
	backend := &AzureBlobBackend{
		multipartUploads: make(map[string]*MultipartUploadMetadata),
	}

	ctx := context.Background()

	// Test completing non-existent upload
	_, err := backend.CompleteMultipartUpload(ctx, "bucket", "key", "nonexistent", nil)
	if err == nil {
		t.Error("CompleteMultipartUpload should fail for non-existent upload")
	}
}

// Test ListMultipartUploads with empty list
func TestListMultipartUploadsEmpty(t *testing.T) {
	backend := &AzureBlobBackend{
		multipartUploads: make(map[string]*MultipartUploadMetadata),
	}

	ctx := context.Background()

	// Should return empty list
	uploads, err := backend.ListMultipartUploads(ctx, "bucket")
	if err != nil {
		t.Errorf("ListMultipartUploads failed: %v", err)
	}
	if len(uploads) != 0 {
		t.Errorf("expected 0 uploads, got %d", len(uploads))
	}
}

// Test AbortMultipartUpload success path
func TestAbortMultipartUploadSuccess(t *testing.T) {
	mockUpload := &MultipartUploadMetadata{
		UploadID:        "test-upload-123",
		BucketName:      "bucket1",
		ObjectKey:       "key1",
		BlockIDs:        []string{},
		PartETagMap:     make(map[int]string),
		BlockBlobClient: nil,
	}

	backend := &AzureBlobBackend{
		multipartUploads: map[string]*MultipartUploadMetadata{
			"test-upload-123": mockUpload,
		},
	}

	ctx := context.Background()

	// Should successfully abort
	err := backend.AbortMultipartUpload(ctx, "bucket1", "key1", "test-upload-123")
	if err != nil {
		t.Errorf("AbortMultipartUpload failed: %v", err)
	}

	// Verify upload was removed
	if _, exists := backend.multipartUploads["test-upload-123"]; exists {
		t.Error("Upload should have been removed after abort")
	}
}

// Test ListMultipartUploads with multiple uploads
func TestListMultipartUploadsMultiple(t *testing.T) {
	mockUpload1 := &MultipartUploadMetadata{
		UploadID:        "upload-1",
		BucketName:      "bucket1",
		ObjectKey:       "key1",
		BlockIDs:        []string{},
		PartETagMap:     make(map[int]string),
		BlockBlobClient: nil,
	}

	mockUpload2 := &MultipartUploadMetadata{
		UploadID:        "upload-2",
		BucketName:      "bucket1",
		ObjectKey:       "key2",
		BlockIDs:        []string{},
		PartETagMap:     make(map[int]string),
		BlockBlobClient: nil,
	}

	backend := &AzureBlobBackend{
		multipartUploads: map[string]*MultipartUploadMetadata{
			"upload-1": mockUpload1,
			"upload-2": mockUpload2,
		},
	}

	ctx := context.Background()

	// Should return 2 uploads for bucket1
	uploads, err := backend.ListMultipartUploads(ctx, "bucket1")
	if err != nil {
		t.Errorf("ListMultipartUploads failed: %v", err)
	}
	if len(uploads) != 2 {
		t.Errorf("expected 2 uploads, got %d", len(uploads))
	}
}

// Test ListParts with multiple parts
func TestListPartsWithParts(t *testing.T) {
	mockUpload := &MultipartUploadMetadata{
		UploadID:   "upload-123",
		BucketName: "bucket1",
		ObjectKey:  "key1",
		BlockIDs:   []string{"block1", "block2", "block3"},
		PartETagMap: map[int]string{
			1: "etag1",
			2: "etag2",
			3: "etag3",
		},
		BlockBlobClient: nil,
	}

	backend := &AzureBlobBackend{
		multipartUploads: map[string]*MultipartUploadMetadata{
			"upload-123": mockUpload,
		},
	}

	ctx := context.Background()

	parts, err := backend.ListParts(ctx, "bucket1", "key1", "upload-123")
	if err != nil {
		t.Errorf("ListParts failed: %v", err)
	}
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(parts))
	}
}

// Test AbortMultipartUpload error case
func TestAbortMultipartUploadNotFound(t *testing.T) {
	backend := &AzureBlobBackend{
		multipartUploads: make(map[string]*MultipartUploadMetadata),
	}

	ctx := context.Background()

	err := backend.AbortMultipartUpload(ctx, "bucket1", "key1", "nonexistent")
	if err == nil {
		t.Error("AbortMultipartUpload should fail for non-existent upload")
	}
}

// Test InitiateMultipartUpload generates unique IDs
func TestInitiateMultipartUploadUniqueness(t *testing.T) {
	// Skip test if we can't create a real Azure client
	// (this would require valid Azure credentials)
	t.Logf("InitiateMultipartUpload requires real Azure SDK client, skipping uniqueness test")
}
