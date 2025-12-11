package config

import (
	"os"
	"testing"
)

func TestAzureAuthConfigAccountKey(t *testing.T) {
	// Set up environment
	_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
	_ = os.Setenv("AZURE_STORAGE_KEY", "dGVzdGtleQ==")
	defer func() { _ = os.Unsetenv("AZURE_STORAGE_ACCOUNT") }()
	defer func() { _ = os.Unsetenv("AZURE_STORAGE_KEY") }()

	cfg, err := LoadAzureAuthConfig()
	if err != nil {
		t.Errorf("LoadAzureAuthConfig() failed: %v", err)
	}

	if cfg.Mode != AuthModeAccountKey {
		t.Errorf("Expected mode %s, got %s", AuthModeAccountKey, cfg.Mode)
	}

	if cfg.StorageAccountName != "testaccount" {
		t.Errorf("Expected account name 'testaccount', got '%s'", cfg.StorageAccountName)
	}

	if cfg.AccountKey != "dGVzdGtleQ==" {
		t.Errorf("Expected account key 'dGVzdGtleQ==', got '%s'", cfg.AccountKey)
	}
}

func TestAzureAuthConfigSAS(t *testing.T) {
	_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
	_ = os.Setenv("AZURE_STORAGE_SAS_TOKEN", "sv=2021-06-08&st=2023-01-01&se=2024-01-01&sr=c&sp=racwd")
	defer func() { _ = os.Unsetenv("AZURE_STORAGE_ACCOUNT") }()
	defer func() { _ = os.Unsetenv("AZURE_STORAGE_SAS_TOKEN") }()
	// Unset account key if set from previous test
	_ = os.Unsetenv("AZURE_STORAGE_KEY")

	cfg, err := LoadAzureAuthConfig()
	if err != nil {
		t.Errorf("LoadAzureAuthConfig() failed: %v", err)
	}

	if cfg.Mode != AuthModeSAS {
		t.Errorf("Expected mode %s, got %s", AuthModeSAS, cfg.Mode)
	}

	if cfg.SASToken != "sv=2021-06-08&st=2023-01-01&se=2024-01-01&sr=c&sp=racwd" {
		t.Errorf("Unexpected SAS token")
	}
}

func TestAzureAuthConfigServicePrincipal(t *testing.T) {
	_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
	_ = os.Setenv("AZURE_TENANT_ID", "00000000-0000-0000-0000-000000000000")
	_ = os.Setenv("AZURE_CLIENT_ID", "00000000-0000-0000-0000-000000000001")
	_ = os.Setenv("AZURE_CLIENT_SECRET", "test-secret")
	defer func() { _ = os.Unsetenv("AZURE_STORAGE_ACCOUNT") }()
	defer func() { _ = os.Unsetenv("AZURE_TENANT_ID") }()
	defer func() { _ = os.Unsetenv("AZURE_CLIENT_ID") }()
	defer func() { _ = os.Unsetenv("AZURE_CLIENT_SECRET") }()
	// Unset other auth methods
	_ = os.Unsetenv("AZURE_STORAGE_KEY")
	_ = os.Unsetenv("AZURE_STORAGE_SAS_TOKEN")

	cfg, err := LoadAzureAuthConfig()
	if err != nil {
		t.Errorf("LoadAzureAuthConfig() failed: %v", err)
	}

	if cfg.Mode != AuthModeSPN {
		t.Errorf("Expected mode %s, got %s", AuthModeSPN, cfg.Mode)
	}

	if cfg.SPNClientID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("Unexpected client ID")
	}

	if cfg.TenantID != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("Unexpected tenant ID")
	}
}

func TestAzureAuthConfigMSI(t *testing.T) {
	_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
	_ = os.Setenv("AZURE_USE_MSI", "true")
	defer func() { _ = os.Unsetenv("AZURE_STORAGE_ACCOUNT") }()
	defer func() { _ = os.Unsetenv("AZURE_USE_MSI") }()
	// Unset other auth methods
	_ = os.Unsetenv("AZURE_STORAGE_KEY")
	_ = os.Unsetenv("AZURE_STORAGE_SAS_TOKEN")
	_ = os.Unsetenv("AZURE_CLIENT_ID")
	_ = os.Unsetenv("AZURE_CLIENT_SECRET")
	_ = os.Unsetenv("AZURE_TENANT_ID")

	cfg, err := LoadAzureAuthConfig()
	if err != nil {
		t.Errorf("LoadAzureAuthConfig() failed: %v", err)
	}

	if cfg.Mode != AuthModeMSI {
		t.Errorf("Expected mode %s, got %s", AuthModeMSI, cfg.Mode)
	}
}

func TestAzureAuthConfigValidation(t *testing.T) {
	tests := []struct {
		name       string
		mode       AzureAuthMode
		setupFn    func()
		cleanupFn  func()
		shouldFail bool
	}{
		{
			name: "account_key_valid",
			mode: AuthModeAccountKey,
			setupFn: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Setenv("AZURE_STORAGE_KEY", "dGVzdGtleQ==")
			},
			cleanupFn: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
			},
			shouldFail: false,
		},
		{
			name: "account_key_missing",
			mode: AuthModeAccountKey,
			setupFn: func() {
				_ = os.Setenv("AZURE_STORAGE_ACCOUNT", "testaccount")
				_ = os.Unsetenv("AZURE_STORAGE_KEY")
			},
			cleanupFn: func() {
				_ = os.Unsetenv("AZURE_STORAGE_ACCOUNT")
			},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			defer tt.cleanupFn()

			cfg := &AzureAuthConfig{
				Mode:               tt.mode,
				StorageAccountName: "testaccount",
				AccountKey:         "dGVzdGtleQ==",
			}

			err := cfg.Validate()
			if (err != nil) != tt.shouldFail {
				t.Errorf("Validate() error = %v, shouldFail = %v", err, tt.shouldFail)
			}
		})
	}
}

func TestAuthModeString(t *testing.T) {
	tests := []struct {
		mode     AzureAuthMode
		expected string
	}{
		{AuthModeAccountKey, "account_key"},
		{AuthModeSAS, "sas"},
		{AuthModeMSI, "msi"},
		{AuthModeSPN, "spn"},
		{AuthModeFederatedToken, "federated_token"},
		{AuthModeAzCLI, "az_cli"},
	}

	for _, tt := range tests {
		if tt.mode.String() != tt.expected {
			t.Errorf("AuthMode.String() = %s, expected %s", tt.mode.String(), tt.expected)
		}
	}
}

func TestAuthModeProperties(t *testing.T) {
	tests := []struct {
		mode                     AzureAuthMode
		expectedTokenBased       bool
		expectedServicePrincipal bool
	}{
		{AuthModeAccountKey, false, false},
		{AuthModeSAS, true, false},
		{AuthModeMSI, false, false},
		{AuthModeSPN, false, true},
		{AuthModeFederatedToken, true, true},
		{AuthModeAzCLI, false, false},
	}

	for _, tt := range tests {
		if tt.mode.IsTokenBased() != tt.expectedTokenBased {
			t.Errorf("%s.IsTokenBased() = %v, expected %v", tt.mode, tt.mode.IsTokenBased(), tt.expectedTokenBased)
		}

		if tt.mode.RequiresServicePrincipal() != tt.expectedServicePrincipal {
			t.Errorf("%s.RequiresServicePrincipal() = %v, expected %v", tt.mode, tt.mode.RequiresServicePrincipal(), tt.expectedServicePrincipal)
		}
	}
}
