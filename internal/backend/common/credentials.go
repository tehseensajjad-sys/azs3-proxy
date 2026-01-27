package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/config"
)

// CredentialProvider abstracts Azure credential creation for blob and files.
type CredentialProvider interface {
	GetCredential(ctx context.Context) (azcore.TokenCredential, error)
	String() string
}

// AccountKeyCredential handles account key authentication.
type AccountKeyCredential struct {
	AccountName string
	AccountKey  string
}

func NewAccountKeyCredential(accountName, accountKey string) *AccountKeyCredential {
	return &AccountKeyCredential{AccountName: accountName, AccountKey: accountKey}
}

func (c *AccountKeyCredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	return nil, fmt.Errorf("account key auth uses shared key, not token credential")
}

func (c *AccountKeyCredential) String() string {
	return fmt.Sprintf("AccountKey(account=%s)", c.AccountName)
}

// SASTokenCredential handles SAS token authentication.
type SASTokenCredential struct {
	SASToken string
}

func NewSASTokenCredential(sasToken string) *SASTokenCredential {
	return &SASTokenCredential{SASToken: sasToken}
}

func (c *SASTokenCredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	return nil, fmt.Errorf("SAS token auth uses URL parameters, not token credential")
}

func (c *SASTokenCredential) String() string {
	return "SASToken"
}

// ManagedIdentityCredential handles MSI authentication.
type ManagedIdentityCredential struct {
	ClientID string
}

func NewManagedIdentityCredential(clientID string) *ManagedIdentityCredential {
	return &ManagedIdentityCredential{ClientID: clientID}
}

func (c *ManagedIdentityCredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	return nil, fmt.Errorf("MSI credential requires azure-identity SDK")
}

func (c *ManagedIdentityCredential) String() string {
	if c.ClientID != "" {
		return fmt.Sprintf("ManagedIdentity(clientID=%s)", c.ClientID)
	}
	return "ManagedIdentity(system)"
}

// ServicePrincipalCredential handles SPN authentication.
type ServicePrincipalCredential struct {
	TenantID     string
	ClientID     string
	ClientSecret string
}

func NewServicePrincipalCredential(tenantID, clientID, clientSecret string) *ServicePrincipalCredential {
	return &ServicePrincipalCredential{TenantID: tenantID, ClientID: clientID, ClientSecret: clientSecret}
}

func (c *ServicePrincipalCredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	return nil, fmt.Errorf("SPN credential requires azure-identity SDK")
}

func (c *ServicePrincipalCredential) String() string {
	return fmt.Sprintf("ServicePrincipal(tenant=%s,client=%s)", c.TenantID, c.ClientID)
}

// FederatedTokenCredential handles OIDC federated token authentication.
type FederatedTokenCredential struct {
	TenantID  string
	ClientID  string
	TokenFile string
}

func NewFederatedTokenCredential(tenantID, clientID, tokenFile string) *FederatedTokenCredential {
	return &FederatedTokenCredential{TenantID: tenantID, ClientID: clientID, TokenFile: tokenFile}
}

func (c *FederatedTokenCredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	return nil, fmt.Errorf("federated token credential requires azure-identity SDK")
}

func (c *FederatedTokenCredential) String() string {
	return fmt.Sprintf("FederatedToken(tenant=%s,client=%s)", c.TenantID, c.ClientID)
}

// AzCLICredential handles Azure CLI cached credentials.
type AzCLICredential struct{}

func NewAzCLICredential() *AzCLICredential { return &AzCLICredential{} }

func (c *AzCLICredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	return nil, fmt.Errorf("azure CLI credential requires azure-identity SDK")
}

func (c *AzCLICredential) String() string { return "AzureCLI" }

// NewCredentialProvider creates appropriate credential provider based on auth mode.
func NewCredentialProvider(authConfig *config.AzureAuthConfig, logger *zap.Logger) (CredentialProvider, error) {
	logger.Info("initializing azure credentials", zap.String("auth_mode", authConfig.Mode.String()))

	switch authConfig.Mode {
	case config.AuthModeAccountKey:
		logger.Info("using account key authentication")
		return NewAccountKeyCredential(authConfig.StorageAccountName, authConfig.AccountKey), nil

	case config.AuthModeSAS:
		logger.Info("using SAS token authentication")
		return NewSASTokenCredential(authConfig.SASToken), nil

	case config.AuthModeMSI:
		logger.Info("using managed identity authentication")
		return NewManagedIdentityCredential(authConfig.MSIClientID), nil

	case config.AuthModeSPN:
		logger.Info("using service principal authentication")
		return NewServicePrincipalCredential(authConfig.TenantID, authConfig.SPNClientID, authConfig.SPNClientSecret), nil

	case config.AuthModeFederatedToken:
		logger.Info("using federated token authentication")
		return NewFederatedTokenCredential(authConfig.TenantID, authConfig.FederatedClientID, authConfig.FederatedTokenFile), nil

	case config.AuthModeAzCLI:
		logger.Info("using Azure CLI authentication")
		return NewAzCLICredential(), nil

	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", authConfig.Mode)
	}
}

// HashSecret returns a short hash of sensitive values so they aren't exposed in cache keys or logs.
func HashSecret(value string) string {
	if value == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:8]) // 16 hex chars
}

// BuildCacheKey generates a reusable client cache key for Azure storage clients.
// Hashes secrets to avoid leaking credentials while still differentiating configurations.
func BuildCacheKey(cfg *config.AzureAuthConfig) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		cfg.Mode,
		cfg.StorageAccountName,
		HashSecret(cfg.AccountKey),
		HashSecret(cfg.SASToken),
		cfg.StorageAccountURL,
		cfg.MSIClientID,
		cfg.SPNClientID,
		cfg.FederatedClientID,
	)
}

// ServiceEndpoint builds a default endpoint for the given service (e.g., "blob", "file")
// unless a custom URL is provided.
func ServiceEndpoint(accountName, customURL, serviceHost string) string {
	if customURL != "" {
		return customURL
	}
	return fmt.Sprintf("https://%s.%s.core.windows.net", accountName, serviceHost)
}

// SASURL appends a SAS token to a base URL.
func SASURL(baseURL, sasToken string) string {
	return fmt.Sprintf("%s?%s", baseURL, sasToken)
}
