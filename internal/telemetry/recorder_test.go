package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
)

func TestRecordMetrics(t *testing.T) {
	ctx := context.Background()

	// Test disabled manager
	mgr, _ := NewManager(ctx)
	mgr.RecordS3Request(ctx, "GetObject", true, "")
	mgr.RecordS3Request(ctx, "GetObject", false, "AccessDenied")
	mgr.RecordCacheHit(ctx, "key")
	mgr.RecordCacheMiss(ctx, "key")
	mgr.RecordCacheEviction(ctx, "size")
	mgr.RecordCacheExpiration(ctx)
	mgr.RecordCacheOperation(ctx, "get")
}

func TestRecordMetrics_Enabled(t *testing.T) {
	ctx := context.Background()

	// Manually construct enabled manager
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	mp, err := NewMetricsProvider(ctx, provider)
	if err != nil {
		t.Fatalf("NewMetricsProvider failed: %v", err)
	}

	cfg := &TelemetryConfig{
		Enabled: true,
	}

	mgr := &Manager{
		cfg:             cfg,
		meterProvider:   provider,
		metricsProvider: mp,
	}

	// Record metrics
	mgr.RecordS3Request(ctx, "GetObject", true, "")
	mgr.RecordS3Request(ctx, "GetObject", false, "AccessDenied")
	mgr.RecordS3Request(ctx, "HeadObject", false, "")
	mgr.RecordCacheHit(ctx, "key")
	mgr.RecordCacheMiss(ctx, "key")
	mgr.RecordCacheEviction(ctx, "size")
	mgr.RecordCacheExpiration(ctx)
	mgr.RecordCacheOperation(ctx, "get")
}
