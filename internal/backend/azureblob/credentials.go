package azureblob

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"github.com/vibhansa-msft/azs3-proxy/internal/version"
)

// CredentialProvider abstracts Azure credential creation
type CredentialProvider interface {
	GetCredential(ctx context.Context) (azcore.TokenCredential, error)
	String() string
}

// AccountKeyCredential handles account key authentication
type AccountKeyCredential struct {
	accountName string
	accountKey  string
}

func NewAccountKeyCredential(accountName, accountKey string) *AccountKeyCredential {
	return &AccountKeyCredential{
		accountName: accountName,
		accountKey:  accountKey,
	}
}

func (c *AccountKeyCredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	// Account key doesn't use token-based auth, it returns a special sharedKeyCredential
	// We'll handle this differently in the client factory
	return nil, fmt.Errorf("account key auth uses shared key, not token credential")
}

func (c *AccountKeyCredential) String() string {
	return fmt.Sprintf("AccountKey(account=%s)", c.accountName)
}

// SASTokenCredential handles SAS token authentication
type SASTokenCredential struct {
	sasToken string
}

func NewSASTokenCredential(sasToken string) *SASTokenCredential {
	return &SASTokenCredential{sasToken: sasToken}
}

func (c *SASTokenCredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	return nil, fmt.Errorf("SAS token auth uses URL parameters, not token credential")
}

func (c *SASTokenCredential) String() string {
	return "SASToken"
}

// ManagedIdentityCredential handles MSI authentication
type ManagedIdentityCredential struct {
	clientID string
}

func NewManagedIdentityCredential(clientID string) *ManagedIdentityCredential {
	return &ManagedIdentityCredential{clientID: clientID}
}

func (c *ManagedIdentityCredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	// Try to get MSI credential from Azure SDK
	// This will use IMDS endpoint or managed identity
	// Note: azure-identity SDK needs to be properly imported
	// For now, returning error to be implemented with proper azure-identity import
	return nil, fmt.Errorf("MSI credential requires azure-identity SDK")
}

func (c *ManagedIdentityCredential) String() string {
	if c.clientID != "" {
		return fmt.Sprintf("ManagedIdentity(clientID=%s)", c.clientID)
	}
	return "ManagedIdentity(system)"
}

// ServicePrincipalCredential handles SPN authentication
type ServicePrincipalCredential struct {
	tenantID     string
	clientID     string
	clientSecret string
}

func NewServicePrincipalCredential(tenantID, clientID, clientSecret string) *ServicePrincipalCredential {
	return &ServicePrincipalCredential{
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (c *ServicePrincipalCredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	// Will use azure-identity ClientSecretCredential
	// Placeholder for now
	return nil, fmt.Errorf("SPN credential requires azure-identity SDK")
}

func (c *ServicePrincipalCredential) String() string {
	return fmt.Sprintf("ServicePrincipal(tenant=%s,client=%s)", c.tenantID, c.clientID)
}

// FederatedTokenCredential handles OIDC federated token authentication
type FederatedTokenCredential struct {
	tenantID  string
	clientID  string
	tokenFile string
}

func NewFederatedTokenCredential(tenantID, clientID, tokenFile string) *FederatedTokenCredential {
	return &FederatedTokenCredential{
		tenantID:  tenantID,
		clientID:  clientID,
		tokenFile: tokenFile,
	}
}

func (c *FederatedTokenCredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	// Will use azure-identity WorkloadIdentityCredential
	return nil, fmt.Errorf("federated token credential requires azure-identity SDK")
}

func (c *FederatedTokenCredential) String() string {
	return fmt.Sprintf("FederatedToken(tenant=%s,client=%s)", c.tenantID, c.clientID)
}

// AzCLICredential handles Azure CLI cached credentials
type AzCLICredential struct{}

func NewAzCLICredential() *AzCLICredential {
	return &AzCLICredential{}
}

func (c *AzCLICredential) GetCredential(ctx context.Context) (azcore.TokenCredential, error) {
	// Will use azure-identity AzureCLICredential
	return nil, fmt.Errorf("azure CLI credential requires azure-identity SDK")
}

func (c *AzCLICredential) String() string {
	return "AzureCLI"
}

// NewCredentialProvider creates appropriate credential provider based on auth mode
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

// BuildClientFromCredential creates an Azure Blob Client from credentials
// This abstracts the different authentication methods
func BuildClientFromCredential(ctx context.Context, authConfig *config.AzureAuthConfig, logger *zap.Logger) (*azblob.Client, error) {
	provider, err := NewCredentialProvider(authConfig, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("building azure blob client", zap.String("auth_mode", provider.String()), zap.String("account", authConfig.StorageAccountName))

	// Configure Azure SDK logging
	// Set global listener for Azure SDK logs
	// Note: This is a global setting, so it affects all Azure clients in the process
	log.SetListener(func(event log.Event, s string) {
		logger.Debug("Azure SDK", zap.String("event", string(event)), zap.String("msg", s))
	})

	// Enable logging of HTTP requests/responses including body
	// We enable Request and Response events
	log.SetEvents(log.EventRequest, log.EventResponse)

	clientOptions := &azblob.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: &http.Client{
				Transport: otelhttp.NewTransport(
					http.DefaultTransport,
					otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
						return "AzureBlob: " + r.Method + " " + r.URL.Path
					}),
				),
			},
			Telemetry: policy.TelemetryOptions{
				ApplicationID: "azs3-proxy/" + version.Version,
			},
			Logging: policy.LogOptions{
				IncludeBody:        true,
				AllowedHeaders:     []string{"*"},
				AllowedQueryParams: []string{"*"},
			},
		},
	}

	// Handle different credential types
	switch cred := provider.(type) {
	case *AccountKeyCredential:
		// Check if a custom storage URL is provided (e.g. for Azurite or Azure Stack)
		if authConfig.StorageAccountURL != "" {
			credential, err := azblob.NewSharedKeyCredential(cred.accountName, cred.accountKey)
			if err != nil {
				return nil, fmt.Errorf("failed to create shared key credential: %w", err)
			}
			client, err := azblob.NewClientWithSharedKeyCredential(authConfig.StorageAccountURL, credential, clientOptions)
			if err != nil {
				return nil, fmt.Errorf("failed to create blob client with shared key: %w", err)
			}
			logger.Info("azure blob client created successfully with custom URL",
				zap.String("auth_mode", cred.String()),
				zap.String("url", authConfig.StorageAccountURL))
			return client, nil
		}

		// Account key uses connection string
		connectionString := fmt.Sprintf(
			"DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=core.windows.net",
			cred.accountName, cred.accountKey)
		client, err := azblob.NewClientFromConnectionString(connectionString, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob client from connection string: %w", err)
		}
		logger.Info("azure blob client created successfully", zap.String("auth_mode", cred.String()))
		return client, nil

	case *SASTokenCredential:
		// SAS token uses URL with token
		storageURL := authConfig.StorageAccountURL
		if authConfig.StorageAccountURL == "" {
			storageURL = fmt.Sprintf("https://%s.blob.core.windows.net", authConfig.StorageAccountName)
		}
		sasURL := fmt.Sprintf("%s?%s", storageURL, cred.sasToken)
		client, err := azblob.NewClientWithNoCredential(sasURL, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob client with SAS token: %w", err)
		}
		logger.Info("azure blob client created successfully", zap.String("auth_mode", cred.String()))
		return client, nil

	case *ManagedIdentityCredential, *ServicePrincipalCredential, *FederatedTokenCredential, *AzCLICredential:
		// These require token credentials from azure-identity SDK
		// For now, we return an error indicating they need to be implemented
		tokenCred, err := cred.GetCredential(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get token credential: %w", err)
		}

		storageURL := authConfig.StorageAccountURL
		if storageURL == "" {
			storageURL = fmt.Sprintf("https://%s.blob.core.windows.net", authConfig.StorageAccountName)
		}

		client, err := azblob.NewClient(storageURL, tokenCred, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob client with token credential: %w", err)
		}
		logger.Info("azure blob client created successfully", zap.String("auth_mode", provider.String()))
		return client, nil

	default:
		return nil, fmt.Errorf("unknown credential type: %T", cred)
	}
}
