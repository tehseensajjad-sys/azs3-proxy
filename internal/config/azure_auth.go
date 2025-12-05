package config

import (
	"fmt"
	"os"
)

// AzureAuthMode represents the authentication method for Azure
type AzureAuthMode string

const (
	// AuthModeAccountKey uses storage account name and key
	AuthModeAccountKey AzureAuthMode = "account_key"
	// AuthModeSAS uses Shared Access Signature token
	AuthModeSAS AzureAuthMode = "sas"
	// AuthModeMSI uses Managed Service Identity
	AuthModeMSI AzureAuthMode = "msi"
	// AuthModeSPN uses Service Principal with client secret
	AuthModeSPN AzureAuthMode = "spn"
	// AuthModeFederatedToken uses OpenID Connect federated token
	AuthModeFederatedToken AzureAuthMode = "federated_token"
	// AuthModeAzCLI uses Azure CLI cached credentials
	AuthModeAzCLI AzureAuthMode = "az_cli"
)

// AzureAuthConfig holds Azure authentication configuration
type AzureAuthConfig struct {
	// Common
	Mode               AzureAuthMode
	SubscriptionID     string
	TenantID           string
	StorageAccountName string
	StorageAccountURL  string

	// Account Key auth
	AccountKey string

	// SAS auth
	SASToken string

	// MSI auth
	MSIClientID string // Optional client ID for user-assigned MSI

	// SPN auth
	SPNClientID     string
	SPNClientSecret string
	SPNObjectID     string

	// Federated Token auth
	FederatedTokenFile string
	FederatedClientID  string

	// AzCLI auth (uses default Azure CLI config)
	// No additional fields needed
}

// LoadAzureAuthConfig loads Azure authentication configuration from environment variables
func LoadAzureAuthConfig() (*AzureAuthConfig, error) {
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

	// Detect authentication mode
	authMode := detectAuthMode()
	if authMode == "" {
		return nil, fmt.Errorf("no Azure authentication method configured. Please set one of: AZURE_STORAGE_KEY (account key), AZURE_STORAGE_SAS_TOKEN (SAS), AZURE_USE_MSI (MSI), AZURE_CLIENT_ID+AZURE_CLIENT_SECRET (SPN), AZURE_FEDERATED_TOKEN_FILE (federated), or AZURE_USE_CLI_AUTH (Azure CLI)")
	}

	cfg.Mode = authMode

	// Mode-specific validation
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Build storage account URL if not provided
	if cfg.StorageAccountURL == "" {
		cfg.StorageAccountURL = fmt.Sprintf("https://%s.blob.core.windows.net", cfg.StorageAccountName)
	}

	return cfg, nil
}

// detectAuthMode detects which authentication mode to use based on environment variables
func detectAuthMode() AzureAuthMode {
	// Check in order of precedence

	// 1. Account Key (most explicit and simple)
	if getEnv("AZURE_STORAGE_KEY", "") != "" {
		return AuthModeAccountKey
	}

	// 2. SAS Token
	if getEnv("AZURE_STORAGE_SAS_TOKEN", "") != "" {
		return AuthModeSAS
	}

	// 3. Federated Token (for OIDC)
	if getEnv("AZURE_FEDERATED_TOKEN_FILE", "") != "" {
		return AuthModeFederatedToken
	}

	// 4. Service Principal (client secret)
	if getEnv("AZURE_CLIENT_ID", "") != "" && getEnv("AZURE_CLIENT_SECRET", "") != "" && getEnv("AZURE_TENANT_ID", "") != "" {
		return AuthModeSPN
	}

	// 5. Managed Identity (can be system or user-assigned)
	if getEnv("AZURE_USE_MSI", "") == "true" || getEnv("IMDS_ENDPOINT", "") != "" {
		return AuthModeMSI
	}

	// 6. Azure CLI (default fallback if nothing else is set)
	if getEnv("AZURE_USE_CLI_AUTH", "") == "true" {
		return AuthModeAzCLI
	}

	// If no explicit preference and Azure SDK env vars are present, use MSI as fallback
	// (Azure SDK will try MSI by default)
	if getEnv("AZURE_CLIENT_ID", "") != "" || getEnv("IMDS_ENDPOINT", "") != "" {
		return AuthModeMSI
	}

	return ""
}

// Validate validates the authentication configuration
func (c *AzureAuthConfig) Validate() error {
	if c.StorageAccountName == "" {
		return fmt.Errorf("AZURE_STORAGE_ACCOUNT is required")
	}

	switch c.Mode {
	case AuthModeAccountKey:
		if c.AccountKey = getEnv("AZURE_STORAGE_KEY", ""); c.AccountKey == "" {
			return fmt.Errorf("AZURE_STORAGE_KEY is required for account key authentication")
		}

	case AuthModeSAS:
		if c.SASToken = getEnv("AZURE_STORAGE_SAS_TOKEN", ""); c.SASToken == "" {
			return fmt.Errorf("AZURE_STORAGE_SAS_TOKEN is required for SAS authentication")
		}

	case AuthModeMSI:
		// MSI can work without explicit client ID (system-assigned)
		// but user-assigned MSI requires AZURE_CLIENT_ID
		if c.MSIClientID != "" {
			// Validate format if provided
		}

	case AuthModeSPN:
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
		if c.FederatedTokenFile == "" {
			return fmt.Errorf("AZURE_FEDERATED_TOKEN_FILE is required for federated token authentication")
		}
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
		// It will use the cached credentials from `az login`

	default:
		return fmt.Errorf("unknown authentication mode: %s", c.Mode)
	}

	return nil
}

// String returns a string representation of the auth mode (safe for logging)
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
