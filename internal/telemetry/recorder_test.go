package telemetry

import (
	"context"
	"testing"

	otelmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestInferMethodMappings(t *testing.T) {
	cases := map[string]string{
		"GetObject":               "GET",
		"ListBuckets":             "GET",
		"PutObject":               "PUT",
		"DeleteObject":            "DELETE",
		"HeadObject":              "HEAD",
		"InitiateMultipartUpload": "POST",
		"UnknownOp":               "UNKNOWN",
	}

	for op, want := range cases {
		if got := inferMethod(op); got != want {
			t.Fatalf("inferMethod(%s) = %s; want %s", op, got, want)
		}
	}
}

func TestRecordAzureRequestDoesNotPanic(t *testing.T) {
	ctx := context.Background()

	mp, err := NewMetricsProvider(ctx, sdkmetric.NewMeterProvider())
	if err != nil {
		t.Fatalf("failed to create metrics provider: %v", err)
	}

	m := &Manager{cfg: &TelemetryConfig{Enabled: true}, metricsProvider: mp}
	m.RecordAzureRequest(ctx, "GetObject", true, "")
	m.RecordAzureRequest(ctx, "PutObject", false, "timeout")
}

func TestRecordMetricsBasic(t *testing.T) {
	ctx := context.Background()
	mgr, _ := NewManager(ctx)
	mgr.RecordS3Request(ctx, "GetObject", true, "")
	mgr.RecordCacheHit(ctx, "key")
}

func TestRecordMetrics_Enabled(t *testing.T) {
	ctx := context.Background()
	reader := otelmetric.NewManualReader()
	provider := otelmetric.NewMeterProvider(otelmetric.WithReader(reader))

	mp, err := NewMetricsProvider(ctx, provider)
	if err != nil {
		t.Fatalf("NewMetricsProvider failed: %v", err)
	}

	cfg := &TelemetryConfig{Enabled: true}
	mgr := &Manager{cfg: cfg, meterProvider: provider, metricsProvider: mp}
	mgr.RecordS3Request(ctx, "GetObject", true, "")
	mgr.RecordCacheHit(ctx, "key")
}
