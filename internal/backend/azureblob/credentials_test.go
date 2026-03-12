package azureblob

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	backendcommon "github.com/vibhansa-msft/azs3-proxy/internal/backend/common"
	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"go.uber.org/zap"
)

func TestNewCredentialProviderAccountKey(t *testing.T) {
	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "testkey==",
	}

	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	provider, err := backendcommon.NewCredentialProvider(authConfig, logger)
	if err != nil {
		t.Logf("NewCredentialProvider returned error (expected for test): %v", err)
	}
	if provider == nil {
		t.Error("Expected non-nil credential provider for account key")
	}
}

func TestNewCredentialProviderSAS(t *testing.T) {
	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeSAS,
		StorageAccountName: "testaccount",
		SASToken:           "sv=2021-06-08&st=2023-01-01&se=2024-01-01",
	}

	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	provider, err := backendcommon.NewCredentialProvider(authConfig, logger)
	if err != nil {
		t.Logf("NewCredentialProvider returned error: %v", err)
	}
	if provider == nil {
		t.Error("Expected non-nil credential provider for SAS")
	}
}

func TestNewCredentialProviderMSI(t *testing.T) {
	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeMSI,
		StorageAccountName: "testaccount",
	}

	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	provider, err := backendcommon.NewCredentialProvider(authConfig, logger)
	if err != nil {
		t.Logf("NewCredentialProvider returned error: %v", err)
	}
	if provider == nil {
		t.Error("Expected non-nil credential provider for MSI")
	}
}

func TestNewCredentialProviderSPN(t *testing.T) {
	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeSPN,
		StorageAccountName: "testaccount",
		TenantID:           "tenant-id",
		SPNClientID:        "client-id",
		SPNClientSecret:    "client-secret",
	}

	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	provider, err := backendcommon.NewCredentialProvider(authConfig, logger)
	if err != nil {
		t.Logf("NewCredentialProvider returned error: %v", err)
	}
	if provider == nil {
		t.Error("Expected non-nil credential provider for SPN")
	}
}

func TestNewCredentialProviderFederatedToken(t *testing.T) {
	tokenContent := "test-token-content"
	tmpFile := createTempFile(t, tokenContent)
	defer func() { _ = os.Remove(tmpFile) }()

	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeFederatedToken,
		StorageAccountName: "testaccount",
		TenantID:           "tenant-id",
		SPNClientID:        "client-id",
		FederatedTokenFile: tmpFile,
	}

	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	provider, err := backendcommon.NewCredentialProvider(authConfig, logger)
	if err != nil {
		t.Logf("NewCredentialProvider returned error: %v", err)
	}
	if provider == nil {
		t.Error("Expected non-nil credential provider for federated token")
	}
}

func TestNewCredentialProviderAzCLI(t *testing.T) {
	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeAzCLI,
		StorageAccountName: "testaccount",
	}

	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	provider, err := backendcommon.NewCredentialProvider(authConfig, logger)
	if err != nil {
		t.Logf("NewCredentialProvider returned error: %v", err)
	}
	if provider == nil {
		t.Error("Expected non-nil credential provider for AzCLI")
	}
}

func TestBuildClientFromCredentialAccountKey(t *testing.T) {
	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==",
	}

	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()
	provider, err := backendcommon.NewCredentialProvider(authConfig, logger)
	if err != nil {
		t.Logf("NewCredentialProvider returned error: %v", err)
	}

	if provider != nil {
		_, _ = BuildClientFromCredential(ctx, authConfig, logger, false)
	}
}

func TestBuildClientFromCredentialSAS(t *testing.T) {
	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeSAS,
		StorageAccountName: "testaccount",
		SASToken:           "sv=2021-06-08&st=2023-01-01&se=2024-01-01",
	}

	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()
	provider, err := backendcommon.NewCredentialProvider(authConfig, logger)
	if err != nil {
		t.Logf("NewCredentialProvider returned error: %v", err)
	}

	if provider != nil {
		_, _ = BuildClientFromCredential(ctx, authConfig, logger, false)
	}
}

func TestBuildClientFromCredentialSAS_SetsUserAgentPrefixOnRequest(t *testing.T) {
	uaCh := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uaCh <- r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><EnumerationResults ServiceEndpoint="http://127.0.0.1"><Containers></Containers><NextMarker/></EnumerationResults>`))
	}))
	defer srv.Close()

	authConfig := &config.AzureAuthConfig{
		Mode:               config.AuthModeSAS,
		StorageAccountName: "testaccount",
		StorageAccountURL:  srv.URL,
		SASToken:           "sv=2021-06-08&sig=fake",
	}

	client, err := BuildClientFromCredential(context.Background(), authConfig, zap.NewNop(), false)
	if err != nil {
		t.Fatalf("failed to create blob client: %v", err)
	}

	pager := client.NewListContainersPager(nil)
	if !pager.More() {
		t.Fatalf("expected pager to have at least one page")
	}
	if _, err := pager.NextPage(context.Background()); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	ua := <-uaCh
	if !strings.Contains(ua, "azpartner-azs3proxy/") {
		t.Fatalf("expected User-Agent to contain azpartner-azs3proxy/, got %q", ua)
	}
}

// Helper function to create a temp file
func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "token-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = tmpFile.Close() }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}
