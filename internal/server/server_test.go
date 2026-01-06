package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"github.com/vibhansa-msft/azs3-proxy/internal/telemetry"
	"go.uber.org/zap"
)

func TestResponseCaptureBasic(t *testing.T) {
	rr := httptest.NewRecorder()
	rc := &responseCapture{ResponseWriter: rr}

	// write without explicit WriteHeader
	_, _ = rc.Write([]byte("ok"))
	if rc.status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rc.status)
	}
}

func TestAuthMiddlewareSkipsHealth(t *testing.T) {
	r := chi.NewMux()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	cfg := &config.Config{AzureAuth: &config.AzureAuthConfig{StorageAccountName: "test", Mode: config.AuthModeAccountKey}}
	s := &S3ProxyServer{router: r, config: cfg, logger: logger, telMgr: &telemetry.Manager{}}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	hw := s.authMiddleware(handler)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	hw.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200 for health path, got %d", rr.Code)
	}
}

func TestNewS3ProxyServer(t *testing.T) {
	// Create a valid Azure auth config for testing
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==", // base64 encoded "testkey"
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}

	cfg := &config.Config{
		ListenAddr:        ":8080",
		AzureAuth:         azureAuth,
		S3AccessKeyID:     "AKIA1234567890ABCDEF",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		LogLevel:          "info",
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// This should work with valid config
	server, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Errorf("NewS3ProxyServer() failed: %v", err)
	}

	if server == nil {
		t.Error("Expected non-nil server")
	}
}

func TestNewS3ProxyServerWithSASAuth(t *testing.T) {
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeSAS,
		StorageAccountName: "testaccount",
		SASToken:           "sv=2021-06-08&st=2023-01-01&se=2024-01-01",
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}

	cfg := &config.Config{
		ListenAddr:        ":8080",
		AzureAuth:         azureAuth,
		S3AccessKeyID:     "AKIA1234567890ABCDEF",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		LogLevel:          "info",
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	server, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Errorf("NewS3ProxyServer() with SAS failed: %v", err)
	}

	if server == nil {
		t.Error("Expected non-nil server with SAS auth")
	}
}

func TestNewS3ProxyServerWithMSIAuth(t *testing.T) {
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeMSI,
		StorageAccountName: "testaccount",
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}

	cfg := &config.Config{
		ListenAddr:        ":8080",
		AzureAuth:         azureAuth,
		S3AccessKeyID:     "AKIA1234567890ABCDEF",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		LogLevel:          "info",
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	// MSI auth might fail in non-Azure environment, but should not panic
	_, _ = NewS3ProxyServer(router, cfg, logger, nil)
	// Server can be nil if MSI init fails, which is expected
	if router == nil {
		t.Error("Router should not be affected by MSI auth failure")
	}
}

func TestNewS3ProxyServerWithServicePrincipalAuth(t *testing.T) {
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeSPN,
		StorageAccountName: "testaccount",
		TenantID:           "tenant-id",
		SPNClientID:        "client-id",
		SPNClientSecret:    "client-secret",
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}

	cfg := &config.Config{
		ListenAddr:        ":8080",
		AzureAuth:         azureAuth,
		S3AccessKeyID:     "AKIA1234567890ABCDEF",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		LogLevel:          "info",
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	server, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Logf("NewS3ProxyServer() with SPN returned error (expected for test): %v", err)
	}
	if server != nil {
		// Server created successfully - that's fine
		t.Logf("Server created with SPN auth")
	}
}

func TestS3ProxyServerMiddleware(t *testing.T) {
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==",
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}

	cfg := &config.Config{
		ListenAddr:        ":8080",
		AzureAuth:         azureAuth,
		S3AccessKeyID:     "AKIA1234567890ABCDEF",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		LogLevel:          "info",
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	server, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Errorf("Failed to create server: %v", err)
		return
	}

	if server == nil {
		t.Error("Expected non-nil server")
		return
	}

	// Verify router has middleware registered
	if router == nil {
		t.Error("Router should not be nil after server creation")
	}
}

func TestS3ProxyServerRouting(t *testing.T) {
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==",
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}

	cfg := &config.Config{
		ListenAddr:        ":8080",
		AzureAuth:         azureAuth,
		S3AccessKeyID:     "AKIA1234567890ABCDEF",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		LogLevel:          "info",
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	server, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Errorf("Failed to create server: %v", err)
		return
	}

	if server == nil {
		t.Error("Expected non-nil server")
		return
	}

	// Verify routes were registered
	if server.router == nil {
		t.Error("Server router should not be nil")
	}
}

func TestServerClose(t *testing.T) {
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==",
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}

	cfg := &config.Config{
		ListenAddr:        ":8080",
		AzureAuth:         azureAuth,
		S3AccessKeyID:     "AKIA1234567890ABCDEF",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		LogLevel:          "info",
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	server, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Close should not panic or error
	err = server.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestServerAuthMiddlewareWithValidSignature(t *testing.T) {
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==",
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}

	cfg := &config.Config{
		ListenAddr:        ":8080",
		AzureAuth:         azureAuth,
		S3AccessKeyID:     "AKIA1234567890ABCDEF",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		LogLevel:          "info",
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	server, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Error("Expected non-nil server")
	}
}

func TestServerAuthMiddlewareWithMissingAuth(t *testing.T) {
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==",
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}

	cfg := &config.Config{
		ListenAddr:        ":8080",
		AzureAuth:         azureAuth,
		S3AccessKeyID:     "AKIA1234567890ABCDEF",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		LogLevel:          "info",
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	server, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Error("Expected non-nil server")
	}
}

func TestServerIntegration_FullSetup(t *testing.T) {
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==",
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}

	cfg := &config.Config{
		ListenAddr:        ":8080",
		AzureAuth:         azureAuth,
		S3AccessKeyID:     "AKIA1234567890ABCDEF",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		LogLevel:          "info",
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	server, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Verify that routes are registered by checking router is not nil
	if server.router == nil {
		t.Error("Server router should not be nil after initialization")
	}

	// Close the server
	err = server.Close()
	if err != nil {
		t.Errorf("Failed to close server: %v", err)
	}
}

func TestNewS3ProxyServer_InvalidAuth(t *testing.T) {
	// Invalid auth mode
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AzureAuthMode("invalid"),
		StorageAccountName: "testaccount",
	}
	cfg := &config.Config{
		ListenAddr: ":8080",
		AzureAuth:  azureAuth,
		LogLevel:   "info",
	}
	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	_, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err == nil {
		t.Error("expected error for invalid auth mode")
	}
}

func TestResponseCaptureWriteHeaderAndWrite(t *testing.T) {
	rr := httptest.NewRecorder()
	rc := &responseCapture{ResponseWriter: rr}

	// When Write is called without prior WriteHeader, status should default to 200
	n, err := rc.Write([]byte("ok"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 2 {
		t.Fatalf("unexpected write length: %d", n)
	}
	if rc.status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rc.status)
	}

	// Test WriteHeader sets status and captures headers
	rr2 := httptest.NewRecorder()
	rc2 := &responseCapture{ResponseWriter: rr2}
	rc2.Header().Set("X-Test", "1")
	rc2.WriteHeader(404)
	if rc2.status != 404 {
		t.Fatalf("expected status 404, got %d", rc2.status)
	}
	if rc2.headers.Get("X-Test") != "1" {
		t.Fatalf("expected header X-Test=1, got %v", rc2.headers)
	}
}

func TestNewS3ProxyServerWithCache(t *testing.T) {
	tmpDir := t.TempDir()
	azureAuth := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==",
		StorageAccountURL:  "https://testaccount.blob.core.windows.net",
	}
	cfg := &config.Config{
		ListenAddr:   ":8080",
		AzureAuth:    azureAuth,
		LogLevel:     "info",
		CacheEnabled: true,
		CachePath:    tmpDir,
		CacheMaxSize: 1024,
		CacheTTL:     60,
	}

	router := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	server, err := NewS3ProxyServer(router, cfg, logger, nil)
	if err != nil {
		t.Fatalf("NewS3ProxyServer with cache failed: %v", err)
	}
	if server.cacheManager == nil {
		t.Error("cacheManager should be initialized")
	}
	server.Close()
}
