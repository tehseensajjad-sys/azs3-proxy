package azurefile

import (
	"testing"

	"github.com/vibhansa-msft/azs3-proxy/internal/backend"
)

// TestAzureFileBackendImplementsInterface verifies that AzureFileBackend
// implements the StorageBackend interface at compile time.
func TestAzureFileBackendImplementsInterface(t *testing.T) {
	var _ backend.StorageBackend = (*AzureFileBackend)(nil)
	t.Log("AzureFileBackend successfully implements StorageBackend interface")
}
