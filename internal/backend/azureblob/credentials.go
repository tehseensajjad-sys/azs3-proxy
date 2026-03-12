package azureblob

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"go.uber.org/zap"

	backendcommon "github.com/vibhansa-msft/azs3-proxy/internal/backend/common"
	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"github.com/vibhansa-msft/azs3-proxy/internal/version"
)

// BuildClientFromCredential creates an Azure Blob Client from credentials
// This abstracts the different authentication methods
func BuildClientFromCredential(ctx context.Context, authConfig *config.AzureAuthConfig, logger *zap.Logger, telemetryEnabled bool) (*azblob.Client, error) {
	provider, err := backendcommon.NewCredentialProvider(authConfig, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("building azure blob client", zap.String("auth_mode", provider.String()), zap.String("account", authConfig.StorageAccountName))

	// Configure Azure SDK logging and capture whether bodies should be logged.
	includeBody := backendcommon.ConfigureSDKLogging(logger)

	// Create a custom transport with optimized pooling and optional OTEL instrumentation.
	transport := backendcommon.NewTransport("AzureBlob: ", telemetryEnabled)

	clientOptions := &azblob.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: &http.Client{
				Transport: transport,
			},
			PerCallPolicies: []policy.Policy{backendcommon.RequestIDPolicy{}},
			Telemetry: policy.TelemetryOptions{
				ApplicationID: version.AzureApplicationIDPrefix + version.Version,
			},
			Logging: policy.LogOptions{
				IncludeBody:        includeBody,
				AllowedHeaders:     []string{"*"},
				AllowedQueryParams: []string{"*"},
			},
		},
	}

	// Handle different credential types
	switch cred := provider.(type) {
	case *backendcommon.AccountKeyCredential:
		endpoint := backendcommon.ServiceEndpoint(authConfig.StorageAccountName, authConfig.StorageAccountURL, "blob")
		credential, err := azblob.NewSharedKeyCredential(cred.AccountName, cred.AccountKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create shared key credential: %w", err)
		}
		client, err := azblob.NewClientWithSharedKeyCredential(endpoint, credential, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob client with shared key: %w", err)
		}
		logger.Info("azure blob client created successfully",
			zap.String("auth_mode", cred.String()),
			zap.String("url", endpoint))
		return client, nil

	case *backendcommon.SASTokenCredential:
		// SAS token uses URL with token
		storageURL := backendcommon.ServiceEndpoint(authConfig.StorageAccountName, authConfig.StorageAccountURL, "blob")
		sasURL := backendcommon.SASURL(storageURL, cred.SASToken)
		client, err := azblob.NewClientWithNoCredential(sasURL, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob client with SAS token: %w", err)
		}
		logger.Info("azure blob client created successfully", zap.String("auth_mode", cred.String()))
		return client, nil

	case *backendcommon.ManagedIdentityCredential, *backendcommon.ServicePrincipalCredential, *backendcommon.FederatedTokenCredential, *backendcommon.AzCLICredential:
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
