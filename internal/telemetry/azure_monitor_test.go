package telemetry

import (
	"testing"
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
