package config

import (
	"os"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		setup     func()
		cleanup   func()
		expectErr bool
	}{
		{
			name: "config from environment",
			setup: func() {
				_ = os.Setenv("LISTEN_ADDR", "0.0.0.0:8080")
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				_ = os.Unsetenv("LISTEN_ADDR")
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
			},
			expectErr: false,
		},
		{
			name: "missing required fields",
			setup: func() {
				// Clear all env vars
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
			},
			cleanup: func() {
				// Cleanup is not needed for failure case
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			defer func() {
				if tt.cleanup != nil {
					tt.cleanup()
				}
			}()

			cfg, err := LoadConfig()
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got %v", err)
			}
			if !tt.expectErr && cfg == nil {
				t.Error("Expected non-nil config")
			}
		})
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Setup required env vars
	_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
	_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
	_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
	_ = os.Setenv("S3_SECRET_KEY", "testsecret")
	defer func() {
		_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
		_ = os.Unsetenv("AZURE_STORAGE_KEY")
		_ = os.Unsetenv("S3_ACCESS_KEY")
		_ = os.Unsetenv("S3_SECRET_KEY")
		_ = os.Unsetenv("LISTEN_ADDR")
		_ = os.Unsetenv("LOG_LEVEL")
	}()

	// Clear defaults to test they are set
	_ = os.Unsetenv("LISTEN_ADDR")
	_ = os.Unsetenv("LOG_LEVEL")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.ListenAddr == "" {
		t.Error("Expected ListenAddr to have default value")
	}

	if cfg.LogLevel == "" {
		t.Error("Expected LogLevel to have default value")
	}
}

func TestLoadConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		setup     func()
		cleanup   func()
		expectErr bool
	}{
		{
			name: "with account key auth",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
			},
			expectErr: false,
		},
		{
			name: "with sas token auth",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_SAS_TOKEN", "sv=2021-06-08&st=2023-01-01&se=2024-01-01")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_SAS_TOKEN")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
			},
			expectErr: false,
		},
		{
			name: "with MSI auth",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_USE_MSI", "true")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_USE_MSI")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
			},
			expectErr: false,
		},
		{
			name: "with service principal auth",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_TENANT_ID", "tenant-id")
				_ = os.Setenv("AZURE_CLIENT_ID", "client-id")
				_ = os.Setenv("AZURE_CLIENT_SECRET", "client-secret")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_TENANT_ID")
				_ = os.Unsetenv("AZURE_CLIENT_ID")
				_ = os.Unsetenv("AZURE_CLIENT_SECRET")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			defer func() {
				if tt.cleanup != nil {
					tt.cleanup()
				}
			}()

			cfg, err := LoadConfig()
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got %v", err)
			}
			if !tt.expectErr && cfg == nil {
				t.Error("Expected non-nil config")
			}
		})
	}
}

func TestTLSConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		setup     func()
		cleanup   func()
		expectErr bool
		expectTLS bool
	}{
		{
			name: "tls_enabled_with_valid_paths",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
				_ = os.Setenv("ENABLE_TLS", "true")
				_ = os.Setenv("TLS_CERT_FILE", "/path/to/cert.pem")
				_ = os.Setenv("TLS_KEY_FILE", "/path/to/key.pem")
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
				_ = os.Unsetenv("ENABLE_TLS")
				_ = os.Unsetenv("TLS_CERT_FILE")
				_ = os.Unsetenv("TLS_KEY_FILE")
			},
			expectErr: false,
			expectTLS: true,
		},
		{
			name: "tls_enabled_without_cert_file",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
				_ = os.Setenv("ENABLE_TLS", "true")
				_ = os.Setenv("TLS_KEY_FILE", "/path/to/key.pem")
				_ = os.Unsetenv("TLS_CERT_FILE")
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
				_ = os.Unsetenv("ENABLE_TLS")
				_ = os.Unsetenv("TLS_CERT_FILE")
				_ = os.Unsetenv("TLS_KEY_FILE")
			},
			expectErr: true,
			expectTLS: false,
		},
		{
			name: "tls_disabled_no_cert_required",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
				_ = os.Setenv("ENABLE_TLS", "false")
				_ = os.Unsetenv("TLS_CERT_FILE")
				_ = os.Unsetenv("TLS_KEY_FILE")
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
				_ = os.Unsetenv("ENABLE_TLS")
			},
			expectErr: false,
			expectTLS: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			defer func() {
				if tt.cleanup != nil {
					tt.cleanup()
				}
			}()

			cfg, err := LoadConfig()
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got %v", err)
			}
			if !tt.expectErr && cfg != nil && cfg.EnableTLS != tt.expectTLS {
				t.Errorf("Expected EnableTLS=%v but got %v", tt.expectTLS, cfg.EnableTLS)
			}
		})
	}
}

func TestCacheConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		setup     func()
		cleanup   func()
		expectErr bool
		checkCfg  func(*Config) bool
	}{
		{
			name: "cache_enabled_with_defaults",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
				_ = os.Setenv("CACHE_ENABLED", "true")
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
				_ = os.Unsetenv("CACHE_ENABLED")
				_ = os.Unsetenv("CACHE_PATH")
				_ = os.Unsetenv("CACHE_MAX_SIZE")
				_ = os.Unsetenv("CACHE_TTL")
			},
			expectErr: false,
			checkCfg: func(cfg *Config) bool {
				return cfg.CacheEnabled && cfg.CachePath != "" && cfg.CacheMaxSize > 0 && cfg.CacheTTL > 0
			},
		},
		{
			name: "cache_enabled_with_custom_values",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
				_ = os.Setenv("CACHE_ENABLED", "true")
				_ = os.Setenv("CACHE_PATH", "/custom/cache/path")
				_ = os.Setenv("CACHE_MAX_SIZE", "2147483648") // 2GB
				_ = os.Setenv("CACHE_TTL", "7200")            // 2 hours
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
				_ = os.Unsetenv("CACHE_ENABLED")
				_ = os.Unsetenv("CACHE_PATH")
				_ = os.Unsetenv("CACHE_MAX_SIZE")
				_ = os.Unsetenv("CACHE_TTL")
			},
			expectErr: false,
			checkCfg: func(cfg *Config) bool {
				return cfg.CacheEnabled && cfg.CachePath == "/custom/cache/path" && cfg.CacheMaxSize == 2147483648 && cfg.CacheTTL == 7200
			},
		},
		{
			name: "cache_disabled_no_validation",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
				_ = os.Setenv("CACHE_ENABLED", "false")
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
				_ = os.Unsetenv("CACHE_ENABLED")
			},
			expectErr: false,
			checkCfg: func(cfg *Config) bool {
				return !cfg.CacheEnabled
			},
		},
		{
			name: "cache_enabled_invalid_max_size",
			setup: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
				_ = os.Setenv("S3_ACCESS_KEY", "testaccess")
				_ = os.Setenv("S3_SECRET_KEY", "testsecret")
				_ = os.Setenv("CACHE_ENABLED", "true")
				_ = os.Setenv("CACHE_MAX_SIZE", "0") // Invalid: must be positive
			},
			cleanup: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
				_ = os.Unsetenv("S3_ACCESS_KEY")
				_ = os.Unsetenv("S3_SECRET_KEY")
				_ = os.Unsetenv("CACHE_ENABLED")
				_ = os.Unsetenv("CACHE_MAX_SIZE")
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			defer func() {
				if tt.cleanup != nil {
					tt.cleanup()
				}
			}()

			cfg, err := LoadConfig()
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got %v", err)
			}
			if !tt.expectErr && cfg != nil && tt.checkCfg != nil && !tt.checkCfg(cfg) {
				t.Errorf("Cache configuration check failed for config: %+v", cfg)
			}
		})
	}
}

func TestValidateConfig_InvalidAuthMode(t *testing.T) {
	cfg := &Config{
		ListenAddr: ":8080",
		AzureAuth: &AzureAuthConfig{
			Mode: "invalid_mode",
		},
		S3AccessKeyID:     "test",
		S3SecretAccessKey: "test",
	}

	err := cfg.Validate()
	// Should handle invalid auth mode gracefully
	if err != nil && err.Error() != "" {
		t.Logf("Config validation returned error: %v", err)
	}
}

func TestValidateConfig_EmptyS3Keys(t *testing.T) {
	cfg := &Config{
		ListenAddr: ":8080",
		AzureAuth: &AzureAuthConfig{
			Mode:               AuthModeAccountKey,
			StorageAccountName: "test",
			AccountKey:         "key",
		},
		S3AccessKeyID:     "",
		S3SecretAccessKey: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for empty S3 keys")
	}
}

func TestValidateConfig_ValidListenAddrVariations(t *testing.T) {
	// ListenAddr is not strictly validated, just used as-is
	testCases := []string{
		":8080",
		"0.0.0.0:8080",
		"localhost:9000",
		"",
	}

	for _, addr := range testCases {
		cfg := &Config{
			ListenAddr:       addr,
			AzureBackendType: "blob",
			AzureAuth: &AzureAuthConfig{
				Mode:               AuthModeAccountKey,
				StorageAccountName: "test",
				AccountKey:         "key",
			},
			S3AccessKeyID:     "test",
			S3SecretAccessKey: "test",
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected no error for ListenAddr=%q, got %v", addr, err)
		}
	}
}

func TestValidateConfig_ValidConfig(t *testing.T) {
	cfg := &Config{
		ListenAddr:       ":8080",
		AzureBackendType: "blob",
		AzureAuth: &AzureAuthConfig{
			Mode:               AuthModeAccountKey,
			StorageAccountName: "test",
			AccountKey:         "key",
		},
		S3AccessKeyID:     "test",
		S3SecretAccessKey: "test",
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}
}

// TestValidateAzureAuthConfig_AccountKey tests account key auth validation
func TestValidateAzureAuthConfig_AccountKey(t *testing.T) {
	tests := []struct {
		name       string
		config     *AzureAuthConfig
		setupEnv   func()
		cleanupEnv func()
		expectErr  bool
		errMsg     string
	}{
		{
			name: "valid account key auth",
			config: &AzureAuthConfig{
				Mode:               AuthModeAccountKey,
				StorageAccountName: "testaccount",
			},
			setupEnv: func() {
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey123")
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
			},
			expectErr: false,
		},
		{
			name: "account key missing storage account",
			config: &AzureAuthConfig{
				Mode: AuthModeAccountKey,
			},
			setupEnv: func() {
				_ = os.Setenv("AZURE_STORAGE_KEY", "testkey")
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
			},
			expectErr: true,
			errMsg:    "AZURE_STORAGE_ACCOUNT is required",
		},
		{
			name: "account key missing storage key",
			config: &AzureAuthConfig{
				Mode:               AuthModeAccountKey,
				StorageAccountName: "testaccount",
			},
			setupEnv: func() {
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
			},
			cleanupEnv: func() {},
			expectErr:  true,
			errMsg:     "AZURE_STORAGE_KEY is required for account key authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			defer tt.cleanupEnv()

			err := tt.config.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("expectErr=%v, got=%v, err=%v", tt.expectErr, err != nil, err)
			}
			if tt.expectErr && tt.errMsg != "" && (err == nil || err.Error() != tt.errMsg) {
				t.Errorf("expected error message %q, got %q", tt.errMsg, err)
			}
		})
	}
}

// TestValidateAzureAuthConfig_SAS tests SAS token auth validation
func TestValidateAzureAuthConfig_SAS(t *testing.T) {
	tests := []struct {
		name       string
		config     *AzureAuthConfig
		setupEnv   func()
		cleanupEnv func()
		expectErr  bool
	}{
		{
			name: "valid SAS auth",
			config: &AzureAuthConfig{
				Mode:               AuthModeSAS,
				StorageAccountName: "testaccount",
			},
			setupEnv: func() {
				_ = os.Setenv("AZURE_STORAGE_SAS_TOKEN", "sv=2021-06-08&...")
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("AZURE_STORAGE_SAS_TOKEN")
			},
			expectErr: false,
		},
		{
			name: "SAS missing token",
			config: &AzureAuthConfig{
				Mode:               AuthModeSAS,
				StorageAccountName: "testaccount",
			},
			setupEnv: func() {
				_ = os.Unsetenv("AZURE_STORAGE_SAS_TOKEN")
			},
			cleanupEnv: func() {},
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			defer tt.cleanupEnv()

			err := tt.config.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("expectErr=%v, got=%v", tt.expectErr, err != nil)
			}
		})
	}
}

// TestValidateAzureAuthConfig_MSI tests managed identity auth validation
func TestValidateAzureAuthConfig_MSI(t *testing.T) {
	config := &AzureAuthConfig{
		Mode:               AuthModeMSI,
		StorageAccountName: "testaccount",
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("MSI auth should not require additional setup, got error: %v", err)
	}
}

// TestValidateAzureAuthConfig_SPN tests service principal auth validation
func TestValidateAzureAuthConfig_SPN(t *testing.T) {
	tests := []struct {
		name       string
		config     *AzureAuthConfig
		setupEnv   func()
		cleanupEnv func()
		expectErr  bool
	}{
		{
			name: "valid SPN auth",
			config: &AzureAuthConfig{
				Mode:               AuthModeSPN,
				StorageAccountName: "testaccount",
				SPNClientID:        "client-id",
				SPNClientSecret:    "secret",
				TenantID:           "tenant-id",
			},
			setupEnv:   func() {},
			cleanupEnv: func() {},
			expectErr:  false,
		},
		{
			name: "SPN missing client ID",
			config: &AzureAuthConfig{
				Mode:               AuthModeSPN,
				StorageAccountName: "testaccount",
				SPNClientSecret:    "secret",
				TenantID:           "tenant-id",
			},
			setupEnv:   func() {},
			cleanupEnv: func() {},
			expectErr:  true,
		},
		{
			name: "SPN missing client secret",
			config: &AzureAuthConfig{
				Mode:               AuthModeSPN,
				StorageAccountName: "testaccount",
				SPNClientID:        "client-id",
				TenantID:           "tenant-id",
			},
			setupEnv:   func() {},
			cleanupEnv: func() {},
			expectErr:  true,
		},
		{
			name: "SPN missing tenant ID",
			config: &AzureAuthConfig{
				Mode:               AuthModeSPN,
				StorageAccountName: "testaccount",
				SPNClientID:        "client-id",
				SPNClientSecret:    "secret",
			},
			setupEnv:   func() {},
			cleanupEnv: func() {},
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			defer tt.cleanupEnv()

			err := tt.config.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("expectErr=%v, got=%v", tt.expectErr, err != nil)
			}
		})
	}
}

// TestValidateAzureAuthConfig_FederatedToken tests federated token auth validation
func TestValidateAzureAuthConfig_FederatedToken(t *testing.T) {
	tests := []struct {
		name       string
		config     *AzureAuthConfig
		setupEnv   func()
		cleanupEnv func()
		expectErr  bool
	}{
		{
			name: "federated token missing file",
			config: &AzureAuthConfig{
				Mode:               AuthModeFederatedToken,
				StorageAccountName: "testaccount",
				FederatedClientID:  "client-id",
				TenantID:           "tenant-id",
			},
			setupEnv:   func() {},
			cleanupEnv: func() {},
			expectErr:  true,
		},
		{
			name: "federated token missing client ID",
			config: &AzureAuthConfig{
				Mode:               AuthModeFederatedToken,
				StorageAccountName: "testaccount",
				FederatedTokenFile: "config_test.go",
				TenantID:           "tenant-id",
			},
			setupEnv:   func() {},
			cleanupEnv: func() {},
			expectErr:  true,
		},
		{
			name: "federated token missing tenant ID",
			config: &AzureAuthConfig{
				Mode:               AuthModeFederatedToken,
				StorageAccountName: "testaccount",
				FederatedTokenFile: "config_test.go",
				FederatedClientID:  "client-id",
			},
			setupEnv:   func() {},
			cleanupEnv: func() {},
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			defer tt.cleanupEnv()

			err := tt.config.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("expectErr=%v, got=%v, err=%v", tt.expectErr, err != nil, err)
			}
		})
	}
}

// TestValidateAzureAuthConfig_AzCLI tests Azure CLI auth validation
func TestValidateAzureAuthConfig_AzCLI(t *testing.T) {
	config := &AzureAuthConfig{
		Mode:               AuthModeAzCLI,
		StorageAccountName: "testaccount",
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("AzCLI auth should not require additional setup, got error: %v", err)
	}
}

// TestValidateAzureAuthConfig_UnknownMode tests unknown auth mode
func TestValidateAzureAuthConfig_UnknownMode(t *testing.T) {
	config := &AzureAuthConfig{
		Mode:               "unknown-mode",
		StorageAccountName: "testaccount",
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for unknown auth mode")
	}
}
