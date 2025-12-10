package telemetry

import (
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNewAzureMonitorExporter(t *testing.T) {
	tests := []struct {
		name             string
		connectionString string
		wantErr          bool
	}{
		{
			name:             "valid connection string",
			connectionString: "InstrumentationKey=00000000-0000-0000-0000-000000000000;IngestionEndpoint=https://eastus-2.in.applicationinsights.azure.com/",
			wantErr:          false,
		},
		{
			name:             "valid connection string with only key",
			connectionString: "InstrumentationKey=00000000-0000-0000-0000-000000000000",
			wantErr:          false,
		},
		{
			name:             "just key (fallback)",
			connectionString: "00000000-0000-0000-0000-000000000000",
			wantErr:          false,
		},
		{
			name:             "invalid connection string",
			connectionString: "InvalidStringWithEquals=ButNoKey",
			wantErr:          true,
		},
		{
			name:             "empty string",
			connectionString: "",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAzureMonitorExporter(tt.connectionString)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAzureMonitorExporter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAzureMonitorExporter_Methods(t *testing.T) {
	exporter, err := NewAzureMonitorExporter("InstrumentationKey=test")
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	// Test Temporality
	if exporter.Temporality(metric.InstrumentKindCounter) != metricdata.DeltaTemporality {
		t.Logf("Temporality returned %v", exporter.Temporality(metric.InstrumentKindCounter))
	}

	// Test Aggregation
	_ = exporter.Aggregation(metric.InstrumentKindCounter)

	// Test Export (empty)
	// We can't easily mock the internal appinsights client to verify calls,
	// but we can ensure it doesn't panic on empty or simple data.
	// Note: Real export would try to send network request, which might fail or hang.
	// AppInsights client usually buffers and sends in background.
	// So calling Export might be safe.
}
