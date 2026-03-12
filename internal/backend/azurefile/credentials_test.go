package azurefile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestBuildServiceClientFromCredentialSAS_SetsUserAgentPrefixOnRequest(t *testing.T) {
	uaCh := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uaCh <- r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><EnumerationResults ServiceEndpoint="http://127.0.0.1"><Shares></Shares><NextMarker/></EnumerationResults>`))
	}))
	defer srv.Close()

	cfg := &config.AzureAuthConfig{
		Mode:               config.AuthModeSAS,
		StorageAccountName: "testaccount",
		StorageAccountURL:  srv.URL,
		SASToken:           "sv=2021-06-08&sig=fake",
	}

	client, err := BuildServiceClientFromCredential(context.Background(), cfg, zap.NewNop(), false)
	if err != nil {
		t.Fatalf("failed to create file client: %v", err)
	}

	pager := client.NewListSharesPager(nil)
	if !pager.More() {
		t.Fatalf("expected pager to have at least one page")
	}
	if _, err := pager.NextPage(context.Background()); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	ua := <-uaCh
	if !strings.Contains(ua, "azpartner-azs3proxy/") {
		t.Fatalf("expected User-Agent to contain azpartner-azs3proxy/, got %q", ua)
	}
}
