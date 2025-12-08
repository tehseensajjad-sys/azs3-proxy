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
				os.Setenv("LISTEN_ADDR", "0.0.0.0:8080")
				os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				os.Setenv("AZURE_STORAGE_KEY", "testkey")
				os.Setenv("S3_ACCESS_KEY", "testaccess")
				os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				os.Unsetenv("LISTEN_ADDR")
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("AZURE_STORAGE_KEY")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
			},
			expectErr: false,
		},
		{
			name: "missing required fields",
			setup: func() {
				// Clear all env vars
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
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
	os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
	os.Setenv("AZURE_STORAGE_KEY", "testkey")
	os.Setenv("S3_ACCESS_KEY", "testaccess")
	os.Setenv("S3_SECRET_KEY", "testsecret")
	defer func() {
		os.Unsetenv("AZURE_STORAGE_ACCOUNT")
		os.Unsetenv("AZURE_STORAGE_KEY")
		os.Unsetenv("S3_ACCESS_KEY")
		os.Unsetenv("S3_SECRET_KEY")
		os.Unsetenv("LISTEN_ADDR")
		os.Unsetenv("LOG_LEVEL")
	}()

	// Clear defaults to test they are set
	os.Unsetenv("LISTEN_ADDR")
	os.Unsetenv("LOG_LEVEL")

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
				os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				os.Setenv("AZURE_STORAGE_KEY", "testkey")
				os.Setenv("S3_ACCESS_KEY", "testaccess")
				os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("AZURE_STORAGE_KEY")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
			},
			expectErr: false,
		},
		{
			name: "with sas token auth",
			setup: func() {
				os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				os.Setenv("AZURE_STORAGE_SAS_TOKEN", "sv=2021-06-08&st=2023-01-01&se=2024-01-01")
				os.Setenv("S3_ACCESS_KEY", "testaccess")
				os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("AZURE_STORAGE_SAS_TOKEN")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
			},
			expectErr: false,
		},
		{
			name: "with MSI auth",
			setup: func() {
				os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				os.Setenv("AZURE_USE_MSI", "true")
				os.Setenv("S3_ACCESS_KEY", "testaccess")
				os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("AZURE_USE_MSI")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
			},
			expectErr: false,
		},
		{
			name: "with service principal auth",
			setup: func() {
				os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				os.Setenv("AZURE_TENANT_ID", "tenant-id")
				os.Setenv("AZURE_CLIENT_ID", "client-id")
				os.Setenv("AZURE_CLIENT_SECRET", "client-secret")
				os.Setenv("S3_ACCESS_KEY", "testaccess")
				os.Setenv("S3_SECRET_KEY", "testsecret")
			},
			cleanup: func() {
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("AZURE_TENANT_ID")
				os.Unsetenv("AZURE_CLIENT_ID")
				os.Unsetenv("AZURE_CLIENT_SECRET")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
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
os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
os.Setenv("AZURE_STORAGE_KEY", "testkey")
os.Setenv("S3_ACCESS_KEY", "testaccess")
os.Setenv("S3_SECRET_KEY", "testsecret")
os.Setenv("ENABLE_TLS", "true")
os.Setenv("TLS_CERT_FILE", "/path/to/cert.pem")
os.Setenv("TLS_KEY_FILE", "/path/to/key.pem")
},
cleanup: func() {
os.Unsetenv("AZURE_STORAGE_ACCOUNT")
os.Unsetenv("AZURE_STORAGE_KEY")
os.Unsetenv("S3_ACCESS_KEY")
os.Unsetenv("S3_SECRET_KEY")
os.Unsetenv("ENABLE_TLS")
os.Unsetenv("TLS_CERT_FILE")
os.Unsetenv("TLS_KEY_FILE")
},
expectErr: false,
expectTLS: true,
},
{
name: "tls_enabled_without_cert_file",
setup: func() {
os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
os.Setenv("AZURE_STORAGE_KEY", "testkey")
os.Setenv("S3_ACCESS_KEY", "testaccess")
os.Setenv("S3_SECRET_KEY", "testsecret")
os.Setenv("ENABLE_TLS", "true")
os.Setenv("TLS_KEY_FILE", "/path/to/key.pem")
os.Unsetenv("TLS_CERT_FILE")
},
cleanup: func() {
os.Unsetenv("AZURE_STORAGE_ACCOUNT")
os.Unsetenv("AZURE_STORAGE_KEY")
os.Unsetenv("S3_ACCESS_KEY")
os.Unsetenv("S3_SECRET_KEY")
os.Unsetenv("ENABLE_TLS")
os.Unsetenv("TLS_CERT_FILE")
os.Unsetenv("TLS_KEY_FILE")
},
expectErr: true,
expectTLS: false,
},
{
name: "tls_disabled_no_cert_required",
setup: func() {
os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
os.Setenv("AZURE_STORAGE_KEY", "testkey")
os.Setenv("S3_ACCESS_KEY", "testaccess")
os.Setenv("S3_SECRET_KEY", "testsecret")
os.Setenv("ENABLE_TLS", "false")
os.Unsetenv("TLS_CERT_FILE")
os.Unsetenv("TLS_KEY_FILE")
},
cleanup: func() {
os.Unsetenv("AZURE_STORAGE_ACCOUNT")
os.Unsetenv("AZURE_STORAGE_KEY")
os.Unsetenv("S3_ACCESS_KEY")
os.Unsetenv("S3_SECRET_KEY")
os.Unsetenv("ENABLE_TLS")
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
				os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				os.Setenv("AZURE_STORAGE_KEY", "testkey")
				os.Setenv("S3_ACCESS_KEY", "testaccess")
				os.Setenv("S3_SECRET_KEY", "testsecret")
				os.Setenv("CACHE_ENABLED", "true")
			},
			cleanup: func() {
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("AZURE_STORAGE_KEY")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
				os.Unsetenv("CACHE_ENABLED")
				os.Unsetenv("CACHE_PATH")
				os.Unsetenv("CACHE_MAX_SIZE")
				os.Unsetenv("CACHE_TTL")
			},
			expectErr: false,
			checkCfg: func(cfg *Config) bool {
				return cfg.CacheEnabled && cfg.CachePath != "" && cfg.CacheMaxSize > 0 && cfg.CacheTTL > 0
			},
		},
		{
			name: "cache_enabled_with_custom_values",
			setup: func() {
				os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				os.Setenv("AZURE_STORAGE_KEY", "testkey")
				os.Setenv("S3_ACCESS_KEY", "testaccess")
				os.Setenv("S3_SECRET_KEY", "testsecret")
				os.Setenv("CACHE_ENABLED", "true")
				os.Setenv("CACHE_PATH", "/custom/cache/path")
				os.Setenv("CACHE_MAX_SIZE", "2147483648") // 2GB
				os.Setenv("CACHE_TTL", "7200")            // 2 hours
			},
			cleanup: func() {
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("AZURE_STORAGE_KEY")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
				os.Unsetenv("CACHE_ENABLED")
				os.Unsetenv("CACHE_PATH")
				os.Unsetenv("CACHE_MAX_SIZE")
				os.Unsetenv("CACHE_TTL")
			},
			expectErr: false,
			checkCfg: func(cfg *Config) bool {
				return cfg.CacheEnabled && cfg.CachePath == "/custom/cache/path" && cfg.CacheMaxSize == 2147483648 && cfg.CacheTTL == 7200
			},
		},
		{
			name: "cache_disabled_no_validation",
			setup: func() {
				os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				os.Setenv("AZURE_STORAGE_KEY", "testkey")
				os.Setenv("S3_ACCESS_KEY", "testaccess")
				os.Setenv("S3_SECRET_KEY", "testsecret")
				os.Setenv("CACHE_ENABLED", "false")
			},
			cleanup: func() {
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("AZURE_STORAGE_KEY")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
				os.Unsetenv("CACHE_ENABLED")
			},
			expectErr: false,
			checkCfg: func(cfg *Config) bool {
				return !cfg.CacheEnabled
			},
		},
		{
			name: "cache_enabled_invalid_max_size",
			setup: func() {
				os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				os.Setenv("AZURE_STORAGE_KEY", "testkey")
				os.Setenv("S3_ACCESS_KEY", "testaccess")
				os.Setenv("S3_SECRET_KEY", "testsecret")
				os.Setenv("CACHE_ENABLED", "true")
				os.Setenv("CACHE_MAX_SIZE", "0") // Invalid: must be positive
			},
			cleanup: func() {
				os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				os.Unsetenv("AZURE_STORAGE_KEY")
				os.Unsetenv("S3_ACCESS_KEY")
				os.Unsetenv("S3_SECRET_KEY")
				os.Unsetenv("CACHE_ENABLED")
				os.Unsetenv("CACHE_MAX_SIZE")
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
