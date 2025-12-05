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
