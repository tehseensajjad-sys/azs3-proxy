package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
)

func TestNewMetricsProvider(t *testing.T) {
	ctx := context.Background()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	mp, err := NewMetricsProvider(ctx, provider)
	if err != nil {
		t.Fatalf("NewMetricsProvider failed: %v", err)
	}

	if mp == nil {
		t.Fatal("NewMetricsProvider returned nil")
	}

	if mp.S3RequestsTotal == nil {
		t.Error("S3RequestsTotal is nil")
	}
	if mp.S3RequestsSuccess == nil {
		t.Error("S3RequestsSuccess is nil")
	}
	if mp.S3RequestsErrors == nil {
		t.Error("S3RequestsErrors is nil")
	}
	if mp.S3ErrorsByType == nil {
		t.Error("S3ErrorsByType is nil")
	}
	if mp.CacheHitsTotal == nil {
		t.Error("CacheHitsTotal is nil")
	}
}
