package config

import (
	"fmt"
	"os"
)

// AzureAuthMode represents the authentication method for Azure Blob Storage.
// It determines which credential type and SDK authentication flow to use.
type AzureAuthMode string

const (
	// AuthModeAccountKey uses storage account name and key.
	// Requires: AZURE_STORAGE_ACCOUNT, AZURE_STORAGE_KEY
	AuthModeAccountKey AzureAuthMode = "account_key"

	// AuthModeSAS uses Shared Access Signature token.
	// Requires: AZURE_STORAGE_ACCOUNT, AZURE_STORAGE_SAS_TOKEN
	AuthModeSAS AzureAuthMode = "sas"

	// AuthModeMSI uses Managed Service Identity (system or user-assigned).
	// Optional: AZURE_CLIENT_ID (for user-assigned MSI)
	AuthModeMSI AzureAuthMode = "msi"

	// AuthModeSPN uses Service Principal with client secret.
	// Requires: AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID
	AuthModeSPN AzureAuthMode = "spn"

	// AuthModeFederatedToken uses OpenID Connect federated token.
	// Requires: AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_FEDERATED_TOKEN_FILE
	AuthModeFederatedToken AzureAuthMode = "federated_token"

	// AuthModeAzCLI uses Azure CLI cached credentials.
	// No additional configuration required, uses ~/.azure/credentials
	AuthModeAzCLI AzureAuthMode = "az_cli"
)

// AzureAuthConfig holds all Azure authentication configuration needed to connect to Azure Blob Storage.
// It supports six authentication methods, with fields populated based on the detected authentication mode.
type AzureAuthConfig struct {
	// Common fields used by all authentication modes
	Mode               AzureAuthMode // The authentication method being used
	SubscriptionID     string        // Azure subscription ID (optional)
	TenantID           string        // Azure tenant ID (required for SPN and federated token auth)
	StorageAccountName string        // Azure storage account name (required)
	StorageAccountURL  string        // Full URL to storage account (auto-generated if not provided)

	// Account Key authentication fields
	AccountKey string // Storage account access key

	// SAS authentication fields
	SASToken string // Shared Access Signature token

	// Managed Service Identity (MSI) authentication fields
	MSIClientID string // Optional client ID for user-assigned MSI (not needed for system-assigned)

	// Service Principal (SPN) authentication fields
	SPNClientID     string // Service principal application ID
	SPNClientSecret string // Service principal client secret
	SPNObjectID     string // Service principal object ID (optional)

	// Federated Token authentication fields
	FederatedTokenFile string // Path to OIDC token file
	FederatedClientID  string // Client ID for federated token

	// Azure CLI authentication
	// No additional fields needed - uses ~/.azure/credentials from logged-in user
}

// LoadAzureAuthConfig loads all Azure authentication configuration from environment variables.
// It auto-detects the authentication mode based on which credentials are available,
// validates that all required fields for that mode are present, and builds the storage account URL.
// Returns error if no valid authentication method is configured or required fields are missing.
func LoadAzureAuthConfig() (*AzureAuthConfig, error) {
	// Initialize configuration with all environment variables that might be needed
	cfg := &AzureAuthConfig{
		SubscriptionID:     getEnv("AZURE_SUBSCRIPTION_ID", ""),
		TenantID:           getEnv("AZURE_TENANT_ID", ""),
		StorageAccountName: getEnvRequired("AZURE_STORAGE_ACCOUNT"),
		StorageAccountURL:  getEnv("AZURE_STORAGE_URL", ""),
		MSIClientID:        getEnv("AZURE_CLIENT_ID", ""),
		SPNClientID:        getEnv("AZURE_CLIENT_ID", ""),
		SPNClientSecret:    getEnv("AZURE_CLIENT_SECRET", ""),
		SPNObjectID:        getEnv("AZURE_OBJECT_ID", ""),
		FederatedClientID:  getEnv("AZURE_CLIENT_ID", ""),
		FederatedTokenFile: getEnv("AZURE_FEDERATED_TOKEN_FILE", ""),
	}

	// Detect which authentication mode should be used based on environment variables
	authMode := detectAuthMode()
	if authMode == "" {
		return nil, fmt.Errorf("no Azure authentication method configured. Please set one of: AZURE_STORAGE_KEY (account key), AZURE_STORAGE_SAS_TOKEN (SAS), AZURE_USE_MSI (MSI), AZURE_CLIENT_ID+AZURE_CLIENT_SECRET (SPN), AZURE_FEDERATED_TOKEN_FILE (federated), or AZURE_USE_CLI_AUTH (Azure CLI)")
	}

	cfg.Mode = authMode

	// Validate that all required fields for the detected mode are present
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Build storage account URL if not explicitly provided
	if cfg.StorageAccountURL == "" {
		if cfg.StorageAccountName == "devstoreaccount1" {
			// Default to local Azurite emulator for devstoreaccount1
			cfg.StorageAccountURL = "http://127.0.0.1:10000/devstoreaccount1"
		} else {
			cfg.StorageAccountURL = fmt.Sprintf("https://%s.blob.core.windows.net", cfg.StorageAccountName)
		}
	}

	return cfg, nil
}

// detectAuthMode detects which authentication mode to use based on available environment variables.
// It checks for credentials in priority order: account key, SAS, federated token, SPN, MSI, Azure CLI.
// Returns empty string if no valid authentication method is found.
func detectAuthMode() AzureAuthMode {
	// 1. Check for account key authentication (most explicit and simple method)
	if getEnv("AZURE_STORAGE_KEY", "") != "" {
		return AuthModeAccountKey
	}

	// 2. Check for SAS token authentication
	if getEnv("AZURE_STORAGE_SAS_TOKEN", "") != "" {
		return AuthModeSAS
	}

	// 3. Check for federated token authentication (OIDC)
	if getEnv("AZURE_FEDERATED_TOKEN_FILE", "") != "" {
		return AuthModeFederatedToken
	}

	// 4. Check for Service Principal authentication (client secret)
	if getEnv("AZURE_CLIENT_ID", "") != "" && getEnv("AZURE_CLIENT_SECRET", "") != "" && getEnv("AZURE_TENANT_ID", "") != "" {
		return AuthModeSPN
	}

	// 5. Check for Managed Identity (system or user-assigned)
	if getEnv("AZURE_USE_MSI", "") == "true" || getEnv("IMDS_ENDPOINT", "") != "" {
		return AuthModeMSI
	}

	// 6. Check for Azure CLI authentication
	if getEnv("AZURE_USE_CLI_AUTH", "") == "true" {
		return AuthModeAzCLI
	}

	// Fallback: If any Azure SDK environment variables are present without explicit config, try MSI
	// (Azure SDK will attempt MSI by default if these env vars are set)
	if getEnv("AZURE_CLIENT_ID", "") != "" || getEnv("IMDS_ENDPOINT", "") != "" {
		return AuthModeMSI
	}

	return ""
}

// Validate validates that all required configuration is present for the configured authentication mode.
// It ensures storage account name is set and performs mode-specific validation.
// Returns error if any required fields are missing.
func (c *AzureAuthConfig) Validate() error {
	// Storage account name is required for all authentication modes
	if c.StorageAccountName == "" {
		return fmt.Errorf("AZURE_STORAGE_ACCOUNT is required")
	}

	// Validate fields specific to the authentication mode being used
	switch c.Mode {
	case AuthModeAccountKey:
		// Account key mode requires the storage account access key
		if c.AccountKey = getEnv("AZURE_STORAGE_KEY", ""); c.AccountKey == "" {
			return fmt.Errorf("AZURE_STORAGE_KEY is required for account key authentication")
		}

	case AuthModeSAS:
		// SAS mode requires a Shared Access Signature token
		if c.SASToken = getEnv("AZURE_STORAGE_SAS_TOKEN", ""); c.SASToken == "" {
			return fmt.Errorf("AZURE_STORAGE_SAS_TOKEN is required for SAS authentication")
		}

	case AuthModeMSI:
		// MSI mode works without explicit client ID (uses system-assigned identity)
		// but user-assigned MSI requires AZURE_CLIENT_ID to specify which identity to use

	case AuthModeSPN:
		// Service Principal mode requires three fields: client ID, client secret, and tenant ID
		if c.SPNClientID == "" {
			return fmt.Errorf("AZURE_CLIENT_ID is required for Service Principal authentication")
		}
		if c.SPNClientSecret == "" {
			return fmt.Errorf("AZURE_CLIENT_SECRET is required for Service Principal authentication")
		}
		if c.TenantID == "" {
			return fmt.Errorf("AZURE_TENANT_ID is required for Service Principal authentication")
		}

	case AuthModeFederatedToken:
		// Federated token mode requires token file, client ID, and tenant ID
		if c.FederatedTokenFile == "" {
			return fmt.Errorf("AZURE_FEDERATED_TOKEN_FILE is required for federated token authentication")
		}
		// Verify the token file actually exists and is readable
		if _, err := os.Stat(c.FederatedTokenFile); err != nil {
			return fmt.Errorf("federated token file not readable: %w", err)
		}
		if c.FederatedClientID == "" {
			return fmt.Errorf("AZURE_CLIENT_ID is required for federated token authentication")
		}
		if c.TenantID == "" {
			return fmt.Errorf("AZURE_TENANT_ID is required for federated token authentication")
		}

	case AuthModeAzCLI:
		// Azure CLI authentication doesn't require additional configuration
		// It will automatically use the cached credentials from `az login`

	default:
		return fmt.Errorf("unknown authentication mode: %s", c.Mode)
	}

	return nil
}

// String returns a string representation of the authentication mode (safe for logging).
// This is useful for logging which auth method is being used without exposing credentials.
func (m AzureAuthMode) String() string {
	return string(m)
}

// IsTokenBased returns true if the auth mode uses tokens (SAS, federated, etc.)
func (m AzureAuthMode) IsTokenBased() bool {
	return m == AuthModeSAS || m == AuthModeFederatedToken
}

// RequiresServicePrincipal returns true if the auth mode requires service principal credentials
func (m AzureAuthMode) RequiresServicePrincipal() bool {
	return m == AuthModeSPN || m == AuthModeFederatedToken
}
