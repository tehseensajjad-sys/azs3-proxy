package azurefile

import (
	"context"
	"testing"

	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"go.uber.org/zap"
)

func TestBuildServiceClientFromCredentialAccountKey(t *testing.T) {
	cfg := &config.AzureAuthConfig{
		Mode:               config.AuthModeAccountKey,
		StorageAccountName: "testaccount",
		AccountKey:         "dGVzdGtleQ==", // base64("testkey")
	}
	client, err := BuildServiceClientFromCredential(context.Background(), cfg, zap.NewNop(), false)
	if err != nil {
		t.Fatalf("expected client without error, got %v", err)
	}
	if client == nil {
		t.Fatalf("expected non-nil client")
	}
}

func TestBuildServiceClientFromCredentialSAS(t *testing.T) {
	cfg := &config.AzureAuthConfig{
		Mode:               config.AuthModeSAS,
		StorageAccountName: "testaccount",
		SASToken:           "sv=2021-06-08&sig=fake",
	}
	client, err := BuildServiceClientFromCredential(context.Background(), cfg, zap.NewNop(), false)
	if err != nil {
		t.Fatalf("expected client without error, got %v", err)
	}
	if client == nil {
		t.Fatalf("expected non-nil client")
	}
}

func TestBuildServiceClientFromCredentialUnsupported(t *testing.T) {
	cfg := &config.AzureAuthConfig{
		Mode:               config.AuthModeMSI,
		StorageAccountName: "testaccount",
	}
	if _, err := BuildServiceClientFromCredential(context.Background(), cfg, zap.NewNop(), false); err == nil {
		t.Fatalf("expected error for unsupported token-based auth")
	}
}
