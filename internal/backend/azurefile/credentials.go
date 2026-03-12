package azurefile

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"
	"go.uber.org/zap"

	backendcommon "github.com/vibhansa-msft/azs3-proxy/internal/backend/common"
	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"github.com/vibhansa-msft/azs3-proxy/internal/version"
)

// BuildServiceClientFromCredential creates an Azure File Service Client from credentials
// This abstracts the different authentication methods for Azure Files
func BuildServiceClientFromCredential(ctx context.Context, authConfig *config.AzureAuthConfig, logger *zap.Logger, telemetryEnabled bool) (*service.Client, error) {
	provider, err := backendcommon.NewCredentialProvider(authConfig, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("building azure file service client",
		zap.String("auth_mode", provider.String()),
		zap.String("account", authConfig.StorageAccountName))

	// Configure Azure SDK logging and capture whether bodies should be logged.
	includeBody := backendcommon.ConfigureSDKLogging(logger)

	// Create a custom transport with optimized pooling and optional OTEL instrumentation.
	transport := backendcommon.NewTransport("AzureFile: ", telemetryEnabled)

	clientOptions := &service.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: &http.Client{Transport: transport},
			Telemetry: policy.TelemetryOptions{
				ApplicationID: version.AzureApplicationIDPrefix + version.Version,
			},
			Logging: policy.LogOptions{
				IncludeBody:        includeBody,
				AllowedHeaders:     []string{"*"},
				AllowedQueryParams: []string{"*"},
			},
			PerCallPolicies: []policy.Policy{backendcommon.RequestIDPolicy{}},
		},
	}

	// Handle different authentication methods
	switch cred := provider.(type) {
	case *backendcommon.AccountKeyCredential:
		serviceURL := backendcommon.ServiceEndpoint(authConfig.StorageAccountName, authConfig.StorageAccountURL, "file")
		credential, err := service.NewSharedKeyCredential(cred.AccountName, cred.AccountKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create shared key credential: %w", err)
		}
		client, err := service.NewClientWithSharedKeyCredential(serviceURL, credential, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create file service client: %w", err)
		}
		logger.Info("azure file service client created successfully", zap.String("auth_mode", cred.String()))
		return client, nil

	case *backendcommon.SASTokenCredential:
		storageURL := backendcommon.ServiceEndpoint(authConfig.StorageAccountName, authConfig.StorageAccountURL, "file")
		sasURL := backendcommon.SASURL(storageURL, cred.SASToken)
		client, err := service.NewClientWithNoCredential(sasURL, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create file service client with SAS token: %w", err)
		}
		logger.Info("azure file service client created successfully", zap.String("auth_mode", cred.String()))
		return client, nil

	default:
		return nil, fmt.Errorf("token-based authentication (%s) for Azure Files is not yet fully implemented - please use AccountKey or SAS token authentication", provider.String())
	}
}
