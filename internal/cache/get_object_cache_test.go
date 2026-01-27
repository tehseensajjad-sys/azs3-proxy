package cache

import (
	"context"
	"testing"

	"github.com/vibhansa-msft/azs3-proxy/internal/telemetry"
)

func TestGetObjectFromCacheHitMissWithTelemetry(t *testing.T) {
	tmpDir := t.TempDir()

	cm, err := NewCacheManager(tmpDir, 10*1024*1024, 3600)
	if err != nil {
		t.Fatalf("NewCacheManager failed: %v", err)
	}
	defer func() { _ = cm.Close() }()

	tel, _ := telemetry.NewManager(context.Background())
	cm.SetTelemetryManager(tel)

	// Miss path
	if path, err := cm.GetObjectFromCache("bucket/miss"); err != nil || path != "" {
		t.Fatalf("expected cache miss with empty path, got path=%q err=%v", path, err)
	}

	// Hit path
	if err := cm.CacheObject("bucket/hit", []byte("data")); err != nil {
		t.Fatalf("CacheObject failed: %v", err)
	}
	if path, err := cm.GetObjectFromCache("bucket/hit"); err != nil || path == "" {
		t.Fatalf("expected cache hit with non-empty path, got path=%q err=%v", path, err)
	}
}
