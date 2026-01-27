package azureblob

import (
	"context"
	"testing"

	backendcommon "github.com/vibhansa-msft/azs3-proxy/internal/backend/common"
	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"go.uber.org/zap"
)

func TestFederatedAndAzCLIAndUnknownProvider(t *testing.T) {
	ctx := context.Background()

	// Federated token credential: String() and GetCredential should behave predictably.
	f := backendcommon.NewFederatedTokenCredential("tenant-1", "client-1", "/no/such/file")
	if got := f.String(); got != "FederatedToken(tenant=tenant-1,client=client-1)" {
		t.Errorf("unexpected FederatedToken String(): %s", got)
	}
	if _, err := f.GetCredential(ctx); err == nil {
		t.Error("expected error from FederatedToken GetCredential")
	}

	// Azure CLI credential: String() and GetCredential should return the expected values/errors.
	a := backendcommon.NewAzCLICredential()
	if a.String() != "AzureCLI" {
		t.Errorf("unexpected AzCLICredential String(): %s", a.String())
	}
	if _, err := a.GetCredential(ctx); err == nil {
		t.Error("expected error from AzCLICredential GetCredential")
	}

	// Unknown/unsupported auth mode passed to NewCredentialProvider should return an error.
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	cfg := &config.AzureAuthConfig{
		Mode:               config.AzureAuthMode("unsupported_mode"),
		StorageAccountName: "testaccount",
	}

	if _, err := backendcommon.NewCredentialProvider(cfg, logger); err == nil {
		t.Error("expected error for unsupported auth mode in NewCredentialProvider")
	}
}
