package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// AzureMonitorExporter implements sdkmetric.Exporter to send metrics to Azure Application Insights
type AzureMonitorExporter struct {
	client appinsights.TelemetryClient
}

// NewAzureMonitorExporter creates a new exporter instance
func NewAzureMonitorExporter(connectionString string) (*AzureMonitorExporter, error) {
	// Parse Connection String to get InstrumentationKey
	// Format: "InstrumentationKey=...;IngestionEndpoint=..."
	parts := strings.Split(connectionString, ";")
	var iKey string
	for _, p := range parts {
		if strings.HasPrefix(p, "InstrumentationKey=") {
			iKey = strings.TrimPrefix(p, "InstrumentationKey=")
			break
		}
	}

	if iKey == "" {
		// Fallback: assume the string is the key if it has no semicolons or equals
		if !strings.Contains(connectionString, "=") {
			iKey = connectionString
		} else {
			return nil, fmt.Errorf("invalid connection string: missing InstrumentationKey")
		}
	}

	if iKey == "" {
		return nil, fmt.Errorf("instrumentation key cannot be empty")
	}

	config := appinsights.NewTelemetryConfiguration(iKey)
	// Note: IngestionEndpoint configuration is not trivially supported by the basic NewTelemetryConfiguration
	// but for standard Azure Monitor, the default endpoint is used.

	client := appinsights.NewTelemetryClientFromConfig(config)

	return &AzureMonitorExporter{
		client: client,
	}, nil
}

// Temporality returns DeltaTemporality so we export the change since last export.
// This fits well with AppInsights "TrackMetric" which usually expects a value to be aggregated.
func (e *AzureMonitorExporter) Temporality(kind metric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}

// Aggregation returns the default aggregation
func (e *AzureMonitorExporter) Aggregation(kind metric.InstrumentKind) metric.Aggregation {
	return metric.DefaultAggregationSelector(kind)
}

// Export transforms OTel metrics to AppInsights telemetry and sends them
func (e *AzureMonitorExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, dp := range data.DataPoints {
					e.trackMetric(m.Name, float64(dp.Value), dp.Attributes)
				}
			case metricdata.Sum[float64]:
				for _, dp := range data.DataPoints {
					e.trackMetric(m.Name, dp.Value, dp.Attributes)
				}
			case metricdata.Gauge[int64]:
				for _, dp := range data.DataPoints {
					e.trackMetric(m.Name, float64(dp.Value), dp.Attributes)
				}
			case metricdata.Gauge[float64]:
				for _, dp := range data.DataPoints {
					e.trackMetric(m.Name, dp.Value, dp.Attributes)
				}
				// Histogram support could be added here by calculating sum/count or sending distribution
			}
		}
	}
	return nil
}

func (e *AzureMonitorExporter) trackMetric(name string, value float64, attributes attribute.Set) {
	telemetry := appinsights.NewMetricTelemetry(name, value)

	// Add attributes as properties
	for _, attr := range attributes.ToSlice() {
		telemetry.Properties[string(attr.Key)] = attr.Value.Emit()
	}

	e.client.Track(telemetry)
}

// ForceFlush flushes the AppInsights channel
func (e *AzureMonitorExporter) ForceFlush(ctx context.Context) error {
	channel := e.client.Channel()
	if channel != nil {
		channel.Flush()
	}
	return nil
}

// Shutdown closes the AppInsights channel
func (e *AzureMonitorExporter) Shutdown(ctx context.Context) error {
	channel := e.client.Channel()
	if channel != nil {
		// Close returns a channel that is closed when the flush is complete
		select {
		case <-channel.Close(10 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
