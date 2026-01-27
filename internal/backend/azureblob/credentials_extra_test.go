package azureblob

import (
	"context"
	"testing"

	backendcommon "github.com/vibhansa-msft/azs3-proxy/internal/backend/common"
)

func TestCredentialMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("AccountKeyCredential", func(t *testing.T) {
		c := backendcommon.NewAccountKeyCredential("account", "key")
		if c.String() != "AccountKey(account=account)" {
			t.Errorf("Unexpected String(): %s", c.String())
		}
		if _, err := c.GetCredential(ctx); err == nil {
			t.Error("Expected error from GetCredential")
		}
	})

	t.Run("SASTokenCredential", func(t *testing.T) {
		c := backendcommon.NewSASTokenCredential("token")
		if c.String() != "SASToken" {
			t.Errorf("Unexpected String(): %s", c.String())
		}
		if _, err := c.GetCredential(ctx); err == nil {
			t.Error("Expected error from GetCredential")
		}
	})

	t.Run("ManagedIdentityCredential", func(t *testing.T) {
		c := backendcommon.NewManagedIdentityCredential("client-id")
		if c.String() != "ManagedIdentity(clientID=client-id)" {
			t.Errorf("Unexpected String(): %s", c.String())
		}
		c2 := backendcommon.NewManagedIdentityCredential("")
		if c2.String() != "ManagedIdentity(system)" {
			t.Errorf("Unexpected String(): %s", c2.String())
		}
		if _, err := c.GetCredential(ctx); err == nil {
			t.Error("Expected error from GetCredential (mock)")
		}
	})

	t.Run("ServicePrincipalCredential", func(t *testing.T) {
		c := backendcommon.NewServicePrincipalCredential("tenant", "client", "secret")
		if c.String() != "ServicePrincipal(tenant=tenant,client=client)" {
			t.Errorf("Unexpected String(): %s", c.String())
		}
		// GetCredential might fail or succeed depending on env, but we just check it runs
		_, _ = c.GetCredential(ctx)
	})
}
