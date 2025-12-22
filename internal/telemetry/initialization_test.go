package telemetry

import (
	"context"
	"os"
	"testing"
)

func TestInitializeProviders(t *testing.T) {
	// Set environment variables for test
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	defer os.Unsetenv("OTEL_EXPORTER_OTLP_INSECURE")

	cfg := &TelemetryConfig{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExportType:     "otlp",
		OTLPEndpoint:   "localhost:4317",
	}

	ctx := context.Background()

	t.Run("InitializeLoggerProvider", func(t *testing.T) {
		lp, err := InitializeLoggerProvider(ctx, cfg)
		if err != nil {
			t.Fatalf("InitializeLoggerProvider failed: %v", err)
		}
		if lp == nil {
			t.Error("Expected non-nil LoggerProvider")
		}
	})

	t.Run("InitializeMeterProvider", func(t *testing.T) {
		mp, err := InitializeMeterProvider(ctx, cfg)
		if err != nil {
			t.Fatalf("InitializeMeterProvider failed: %v", err)
		}
		if mp == nil {
			t.Error("Expected non-nil MeterProvider")
		}
	})

	t.Run("InitializeTracerProvider", func(t *testing.T) {
		tp, err := InitializeTracerProvider(ctx, cfg)
		if err != nil {
			t.Fatalf("InitializeTracerProvider failed: %v", err)
		}
		if tp == nil {
			t.Error("Expected non-nil TracerProvider")
		}
	})
}
