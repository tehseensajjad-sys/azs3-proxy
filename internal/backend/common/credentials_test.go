package common

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/vibhansa-msft/azs3-proxy/internal/config"
	"go.uber.org/zap"
)

func TestCredentialStringsAndErrors(t *testing.T) {
	ctx := context.Background()

	ak := NewAccountKeyCredential("acct", "key==")
	if got := ak.String(); got != "AccountKey(account=acct)" {
		t.Fatalf("unexpected AccountKey String(): %s", got)
	}
	if _, err := ak.GetCredential(ctx); err == nil {
		t.Fatalf("expected error for account key token credential")
	}

	sas := NewSASTokenCredential("sig=fake")
	if sas.String() != "SASToken" {
		t.Fatalf("unexpected SASToken String(): %s", sas.String())
	}
	if _, err := sas.GetCredential(ctx); err == nil {
		t.Fatalf("expected error for SAS token credential")
	}

	msi := NewManagedIdentityCredential("client-id")
	if msi.String() != "ManagedIdentity(clientID=client-id)" {
		t.Fatalf("unexpected MSI String(): %s", msi.String())
	}
	if _, err := msi.GetCredential(ctx); err == nil {
		t.Fatalf("expected error for MSI token credential")
	}

	msiSystem := NewManagedIdentityCredential("")
	if msiSystem.String() != "ManagedIdentity(system)" {
		t.Fatalf("unexpected MSI system String(): %s", msiSystem.String())
	}

	spn := NewServicePrincipalCredential("tenant", "client", "secret")
	if spn.String() != "ServicePrincipal(tenant=tenant,client=client)" {
		t.Fatalf("unexpected SPN String(): %s", spn.String())
	}
	if _, err := spn.GetCredential(ctx); err == nil {
		t.Fatalf("expected error for SPN token credential")
	}

	fed := NewFederatedTokenCredential("tenant", "client", "/tmp/token")
	if fed.String() != "FederatedToken(tenant=tenant,client=client)" {
		t.Fatalf("unexpected federated String(): %s", fed.String())
	}
	if _, err := fed.GetCredential(ctx); err == nil {
		t.Fatalf("expected error for federated token credential")
	}

	cli := NewAzCLICredential()
	if cli.String() != "AzureCLI" {
		t.Fatalf("unexpected Azure CLI String(): %s", cli.String())
	}
	if _, err := cli.GetCredential(ctx); err == nil {
		t.Fatalf("expected error for Azure CLI token credential")
	}
}

func TestNewCredentialProvider(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.AzureAuthConfig
		want interface{}
	}{
		{
			name: "account key",
			cfg: &config.AzureAuthConfig{
				Mode:               config.AuthModeAccountKey,
				StorageAccountName: "acct",
				AccountKey:         "key==",
			},
			want: &AccountKeyCredential{},
		},
		{
			name: "sas",
			cfg: &config.AzureAuthConfig{
				Mode:               config.AuthModeSAS,
				StorageAccountName: "acct",
				SASToken:           "sig=fake",
			},
			want: &SASTokenCredential{},
		},
		{
			name: "msi",
			cfg: &config.AzureAuthConfig{
				Mode:               config.AuthModeMSI,
				StorageAccountName: "acct",
				MSIClientID:        "client",
			},
			want: &ManagedIdentityCredential{},
		},
		{
			name: "spn",
			cfg: &config.AzureAuthConfig{
				Mode:               config.AuthModeSPN,
				StorageAccountName: "acct",
				TenantID:           "tenant",
				SPNClientID:        "client",
				SPNClientSecret:    "secret",
			},
			want: &ServicePrincipalCredential{},
		},
		{
			name: "federated",
			cfg: &config.AzureAuthConfig{
				Mode:               config.AuthModeFederatedToken,
				StorageAccountName: "acct",
				TenantID:           "tenant",
				FederatedClientID:  "client",
				FederatedTokenFile: "/tmp/token",
			},
			want: &FederatedTokenCredential{},
		},
		{
			name: "az cli",
			cfg: &config.AzureAuthConfig{
				Mode:               config.AuthModeAzCLI,
				StorageAccountName: "acct",
			},
			want: &AzCLICredential{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			provider, err := NewCredentialProvider(tt.cfg, logger)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if reflect.TypeOf(provider) != reflect.TypeOf(tt.want) {
				t.Fatalf("provider type = %T, want %T", provider, tt.want)
			}
		})
	}

	t.Run("unsupported", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &config.AzureAuthConfig{Mode: config.AzureAuthMode("unknown"), StorageAccountName: "acct"}
		if _, err := NewCredentialProvider(cfg, logger); err == nil {
			t.Fatalf("expected error for unsupported mode")
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	if got := HashSecret("secret-value"); len(got) != 16 {
		t.Fatalf("expected 16 hex chars, got %d", len(got))
	}
	if got := HashSecret(""); got != "" {
		t.Fatalf("expected empty hash for empty input")
	}
	if got := HashSecret("secret-value"); got != HashSecret("secret-value") {
		t.Fatalf("expected deterministic hashing")
	}

	cfg := &config.AzureAuthConfig{
		Mode:               config.AuthModeSAS,
		StorageAccountName: "acct",
		AccountKey:         "key==",
		SASToken:           "sig=fake",
		StorageAccountURL:  "https://example",
		MSIClientID:        "msi",
		SPNClientID:        "spn",
		FederatedClientID:  "fed",
	}
	if got := BuildCacheKey(cfg); got == "" {
		t.Fatalf("expected non-empty cache key")
	}

	if got := ServiceEndpoint("acct", "", "blob"); got != "https://acct.blob.core.windows.net" {
		t.Fatalf("unexpected service endpoint: %s", got)
	}

	if got := SASURL("https://example", "sig=fake"); got != "https://example?sig=fake" {
		t.Fatalf("unexpected SAS url: %s", got)
	}

	cacheKey := BuildCacheKey(cfg)
	if strings.Contains(cacheKey, "key==") || strings.Contains(cacheKey, "sig=fake") {
		t.Fatalf("cache key should not expose secrets: %s", cacheKey)
	}
	if cacheKey != BuildCacheKey(cfg) {
		t.Fatalf("cache key generation should be deterministic")
	}
}
