//go:build !integration

package integration

import "testing"

// Ensures the package is discoverable without the integration build tag.
func TestIntegrationSkipped(t *testing.T) {
    if testing.Short() {
        t.Skip("integration tests skipped in short mode")
    }
    t.Skip("integration tests require -tags=integration; run: go test -tags=integration ./test/integration/...")
}
