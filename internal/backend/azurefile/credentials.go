package azurefile

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"github.com/vibhansa-msft/azs3-proxy/internal/version"
)

// BuildServiceClientFromCredential creates an Azure File Service Client from credentials
// This abstracts the different authentication methods for Azure Files
func BuildServiceClientFromCredential(ctx context.Context, authConfig *config.AzureAuthConfig, logger *zap.Logger) (*service.Client, error) {
	logger.Info("building azure file service client",
		zap.String("auth_mode", authConfig.Mode.String()),
		zap.String("account", authConfig.StorageAccountName))

	// Configure Azure SDK logging
	log.SetListener(func(event log.Event, s string) {
		logger.Debug("Azure SDK", zap.String("event", string(event)), zap.String("msg", s))
	})

	// Enable logging of HTTP requests/responses including body settings
	includeBody := false
	if strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug" {
		includeBody = true
		log.SetEvents(log.EventRequest, log.EventResponse)
	} else {
		log.SetEvents()
	}

	// Create a custom transport with optimized connection pooling settings
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.MaxIdleConns = 1000
	defaultTransport.MaxIdleConnsPerHost = 1000
	defaultTransport.MaxConnsPerHost = 0 // Unlimited
	defaultTransport.IdleConnTimeout = 90 * time.Second
	defaultTransport.DisableCompression = true

	clientOptions := &service.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: &http.Client{
				Transport: otelhttp.NewTransport(
					defaultTransport,
					otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
						return "AzureFile: " + r.Method + " " + r.URL.Path
					}),
				),
			},
			Telemetry: policy.TelemetryOptions{
				ApplicationID: "azs3-proxy/" + version.Version,
			},
			Logging: policy.LogOptions{
				IncludeBody:        includeBody,
				AllowedHeaders:     []string{"*"},
				AllowedQueryParams: []string{"*"},
			},
		},
	}

	// Handle different authentication methods
	switch authConfig.Mode {
	case config.AuthModeAccountKey:
		// Account key authentication using connection string
		if authConfig.StorageAccountURL != "" {
			// Custom URL (e.g., Azurite or Azure Stack)
			credential, err := service.NewSharedKeyCredential(authConfig.StorageAccountName, authConfig.AccountKey)
			if err != nil {
				return nil, fmt.Errorf("failed to create shared key credential: %w", err)
			}
			client, err := service.NewClientWithSharedKeyCredential(authConfig.StorageAccountURL, credential, clientOptions)
			if err != nil {
				return nil, fmt.Errorf("failed to create file service client with shared key: %w", err)
			}
			logger.Info("azure file service client created successfully with custom URL",
				zap.String("auth_mode", "AccountKey"),
				zap.String("url", authConfig.StorageAccountURL))
			return client, nil
		}

		// Standard Azure Files endpoint
		serviceURL := fmt.Sprintf("https://%s.file.core.windows.net", authConfig.StorageAccountName)
		credential, err := service.NewSharedKeyCredential(authConfig.StorageAccountName, authConfig.AccountKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create shared key credential: %w", err)
		}
		client, err := service.NewClientWithSharedKeyCredential(serviceURL, credential, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create file service client: %w", err)
		}
		logger.Info("azure file service client created successfully", zap.String("auth_mode", "AccountKey"))
		return client, nil

	case config.AuthModeSAS:
		// SAS token authentication
		storageURL := authConfig.StorageAccountURL
		if authConfig.StorageAccountURL == "" {
			storageURL = fmt.Sprintf("https://%s.file.core.windows.net", authConfig.StorageAccountName)
		}
		sasURL := fmt.Sprintf("%s?%s", storageURL, authConfig.SASToken)
		client, err := service.NewClientWithNoCredential(sasURL, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create file service client with SAS token: %w", err)
		}
		logger.Info("azure file service client created successfully", zap.String("auth_mode", "SASToken"))
		return client, nil

	case config.AuthModeMSI, config.AuthModeSPN, config.AuthModeFederatedToken, config.AuthModeAzCLI:
		// Token-based authentication methods
		// Note: Azure Files currently has limited support for token-based authentication in the Go SDK.
		// For production use with Azure Files, please use AccountKey or SAS token authentication.
		// Azure Blob Storage supports all authentication methods.
		return nil, fmt.Errorf("token-based authentication (%s) for Azure Files is not yet fully implemented - please use AccountKey or SAS token authentication", authConfig.Mode.String())

	default:
		return nil, fmt.Errorf("unsupported auth mode for Azure Files: %s", authConfig.Mode)
	}
}
