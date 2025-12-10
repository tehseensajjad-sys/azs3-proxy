# OpenTelemetry Integration

The azs3-proxy supports comprehensive metrics and logging export via OpenTelemetry (OTEL), enabling integration with multiple observability backends.

## Features

- **Multi-Backend Support**: Azure Monitor, OTLP protocol, and Prometheus scraping
- **Metrics Export**: S3 API and cache statistics via OTEL metrics protocol
- **Logs Export**: Structured logging integration (foundation for logs export)
- **Configuration-Driven**: Full control via environment variables
- **Zero Impact When Disabled**: Minimal overhead when telemetry is disabled

## Supported Metrics

### S3 API Metrics
- `s3_requests_total` - Total number of S3 API requests (counter)
- `s3_requests_success_total` - Successful S3 API requests (counter)
- `s3_requests_errors_total` - Failed S3 API requests (counter)
- `s3_errors_by_type_total` - S3 errors categorized by error type (counter)

Attributes:
- `operation`: S3 operation name (e.g., "GetObject", "PutObject")
- `error_type`: Error type when request fails (e.g., "AccessDenied", "NoSuchKey")

### Cache Metrics
- `cache_hits_total` - Total cache hits (counter)
- `cache_misses_total` - Total cache misses (counter)
- `cache_evictions_total` - Total cache evictions (counter)
- `cache_expirations_total` - Cache entries that expired (counter)
- `cache_operations_total` - Cache operations (put, get, delete) (counter)

Attributes:
- `key`: Cache key (for hits/misses)
- `reason`: Eviction reason (e.g., "size_limit", "expired")
- `type`: Operation type (e.g., "put", "get", "delete")

## Configuration

Configure telemetry via environment variables:

```bash
# Enable/disable telemetry
TELEMETRY_ENABLED=true                              # Default: true
TELEMETRY_METRICS_ENABLED=true                      # Default: false
TELEMETRY_LOGS_ENABLED=true                         # Default: false

# Service information
SERVICE_NAME=azs3-proxy                             # Default: azs3-proxy
SERVICE_VERSION=1.0.0                               # Default: 1.0.0
ENVIRONMENT=production                              # Default: development

# Export configuration
TELEMETRY_EXPORT_TYPE=azuremonitor                  # Default: azuremonitor
TELEMETRY_EXPORT_INTERVAL=30s                       # Default: 30s
TELEMETRY_EXPORT_TIMEOUT=30s                        # Default: 30s

# Azure Monitor Configuration
AZURE_MONITOR_ENABLED=true                          # Default: false
AZURE_MONITOR_CONNECTION_STRING=...                 # Required if Azure Monitor enabled

# OTLP Configuration (OpenTelemetry Protocol)
OTLP_ENABLED=false                                  # Default: false
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318  # Default: http://localhost:4318
OTEL_EXPORTER_OTLP_HEADERS=                         # Optional headers

# Prometheus Configuration
PROMETHEUS_ENABLED=false                            # Default: false
PROMETHEUS_PORT=8888                                # Default: 8888
PROMETHEUS_PATH=/metrics                            # Default: /metrics

# Advanced configuration
TELEMETRY_LOG_LEVEL=info                            # Default: info
TELEMETRY_BATCH_SIZE=512                            # Default: 512
TELEMETRY_MAX_QUEUE_SIZE=2048                       # Default: 2048
```

## Azure Monitor Integration

### Prerequisites

1. Azure Monitor resource in Azure Portal
2. Instrumentation Key or Connection String

### Setup Steps

1. Obtain your Connection String from Azure Monitor:
```bash
# In Azure Portal: App Insights → Overview → Connection String
export AZURE_MONITOR_CONNECTION_STRING="InstrumentationKey=<your-key>;..."
```

2. Enable metrics export:
```bash
export TELEMETRY_METRICS_ENABLED=true
export AZURE_MONITOR_ENABLED=true
```

3. Metrics will automatically export every 30 seconds (configurable)

### Viewing Metrics in Azure Monitor

1. Go to App Insights → Metrics
2. Select "Custom Metrics"
3. Search for metrics:
   - `s3_requests_total`
   - `cache_hits_total`
   - etc.

### Azure Managed Grafana

To visualize metrics in Azure Managed Grafana:

1. Create Azure Managed Grafana instance
2. Add Azure Monitor as data source
3. Create dashboards using the exported metrics

## OTLP Integration (OpenTelemetry Protocol)

For OTLP exporters (Jaeger, Tempo, etc.):

```bash
export TELEMETRY_METRICS_ENABLED=true
export OTLP_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=http://your-otel-collector:4318
```

## Prometheus Integration

For Prometheus scraping:

```bash
export TELEMETRY_METRICS_ENABLED=true
export PROMETHEUS_ENABLED=true
export PROMETHEUS_PORT=8888
export PROMETHEUS_PATH=/metrics
```

Then configure Prometheus to scrape:
```yaml
scrape_configs:
  - job_name: 's3-proxy'
    static_configs:
      - targets: ['localhost:8888']
    metrics_path: '/metrics'
```

## Usage Example

```go
package main

import (
	"context"
	"github.com/vibhansa-msft/s3-azure-proxy/internal/telemetry"
)

func main() {
	ctx := context.Background()
	
	// Initialize telemetry manager
	telMgr, err := telemetry.NewManager(ctx)
	if err != nil {
		panic(err)
	}
	defer telMgr.Shutdown(ctx)
	
	// Record S3 operation
	telMgr.RecordS3Request(ctx, "GetObject", true, "")
	
	// Record cache operation
	telMgr.RecordCacheHit(ctx, "cache-key")
}
```

## Implementation Details

### Package Structure

- `config.go` - Configuration loading and validation
- `metrics.go` - OTEL metric instruments and provider initialization
- `manager.go` - Lifecycle management for telemetry components
- `recorder.go` - Recording functions for S3 and cache metrics

### Architecture

```
Application Code
    ↓
Telemetry Recorder Functions (RecordS3Request, RecordCacheHit, etc.)
    ↓
OTEL Meters and Counters
    ↓
OTEL MeterProvider
    ↓
Exporters (Azure Monitor, OTLP, Prometheus)
    ↓
Backend (Azure Monitor, Grafana, Prometheus, etc.)
```

## Best Practices

1. **Initialize Early**: Create the telemetry manager during application startup
2. **Graceful Shutdown**: Always call `Shutdown()` in defer to flush metrics
3. **Disable When Not Needed**: Set environment variables to false to minimize overhead
4. **Use Connection Strings**: For Azure Monitor, use connection strings over instrumentation keys
5. **Configure Export Interval**: Balance between metric freshness and network load

## Troubleshooting

### Metrics not appearing in Azure Monitor

1. Verify connection string is correct
2. Check `TELEMETRY_METRICS_ENABLED=true`
3. Check `AZURE_MONITOR_ENABLED=true`
4. Wait at least one export interval (default 30s) for metrics to appear
5. Ensure Azure Monitor resource exists and is accessible

### High network load

Increase `TELEMETRY_EXPORT_INTERVAL`:
```bash
export TELEMETRY_EXPORT_INTERVAL=60s  # Export every 60 seconds instead of 30
```

### Missing metrics

Ensure metrics are being recorded:
```go
// Check if telemetry is enabled
if telMgr.IsEnabled() {
    telMgr.RecordS3Request(ctx, "GetObject", true, "")
}
```

## Testing

Telemetry module includes comprehensive tests:

```bash
go test ./internal/telemetry -v
```

Tests cover:
- Configuration loading and validation
- Metric recording
- Manager lifecycle
- Helper functions (parseBool, parseInt, parseDuration)
