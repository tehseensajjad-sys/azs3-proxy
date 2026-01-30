package azurefile

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azfile/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type fakeError struct{ code string }

func (f fakeError) Error() string     { return "fake" }
func (f fakeError) ErrorCode() string { return f.code }

func TestRecordAzureRequestTracksTelemetryAndLogs(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	backend := &AzureFileBackend{logger: logger}

	ctx := context.WithValue(context.Background(), "requestID", "req-123") //nolint:staticcheck // tests mirror production string keys
	backend.recordAzureRequest(ctx, "ListBuckets", fakeError{code: "FailCode"})

	if logs.Len() != 1 {
		t.Fatalf("expected one log entry, got %d", logs.Len())
	}
	entry := logs.All()[0]
	if entry.Level != zap.ErrorLevel {
		t.Fatalf("expected error log level, got %s", entry.Level)
	}
	if v := entry.ContextMap()["req_id"]; v != "req-123" {
		t.Fatalf("expected req_id to be propagated, got %v", v)
	}
}

func TestRecordAzureRequestSuccessPath(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	backend := &AzureFileBackend{logger: logger}

	backend.recordAzureRequest(context.Background(), "ListBuckets", nil)
	if logs.Len() != 1 || logs.All()[0].Level != zap.DebugLevel {
		t.Fatalf("expected one debug log, got %d with level %v", logs.Len(), logs.All()[0].Level)
	}
}

func TestCloseClearsClientCache(t *testing.T) {
	// Backup and restore global cache to avoid test interference.
	backup := clientCache
	defer func() { clientCache = backup }()

	clientCache = map[string]*service.Client{
		"key": {},
	}

	backend := &AzureFileBackend{}
	if err := backend.Close(); err != nil {
		t.Fatalf("expected nil error closing backend: %v", err)
	}
	if len(clientCache) != 0 {
		t.Fatalf("expected cache to be cleared, got %d entries", len(clientCache))
	}
}
