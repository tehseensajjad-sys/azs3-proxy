# OpenTelemetry Integration

The azs3-proxy supports comprehensive metrics and logging export via OpenTelemetry (OTEL), enabling integration with multiple observability backends.

## Features

- **Multi-Backend Support**: OTLP protocol
- **Metrics Export**: S3 API and cache statistics via OTEL metrics protocol
- **Distributed Tracing**: Request tracing via OTLP (gRPC)
- **Logs Export**: Application logs exported via OTLP
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

# Service information
SERVICE_NAME=azs3-proxy                             # Default: azs3-proxy
SERVICE_VERSION=1.0.0                               # Default: 1.0.0
ENVIRONMENT=production                              # Default: development

# Export configuration
TELEMETRY_EXPORT_TYPE=otlp                        # Default: otlp
TELEMETRY_EXPORT_INTERVAL=30s                       # Default: 30s
TELEMETRY_EXPORT_TIMEOUT=30s                        # Default: 30s

# OTLP Configuration (OpenTelemetry Protocol)
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317          # Default: localhost:4317
OTEL_EXPORTER_OTLP_HEADERS=                         # Optional headers

# Advanced configuration
TELEMETRY_LOG_LEVEL=info                            # Default: info
TELEMETRY_BATCH_SIZE=512                            # Default: 512
TELEMETRY_MAX_QUEUE_SIZE=2048                       # Default: 2048
```

## OTLP Integration (OpenTelemetry Protocol)

For OTLP exporters (Jaeger, Tempo, etc.):

```bash
export TELEMETRY_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=your-otel-collector:4317
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
Exporters (OTLP)
    ↓
Backend (Grafana, Prometheus, etc.)
```

## Best Practices

1. **Initialize Early**: Create the telemetry manager during application startup
2. **Graceful Shutdown**: Always call `Shutdown()` in defer to flush metrics
3. **Disable When Not Needed**: Set environment variables to false to minimize overhead
4. **Configure Export Interval**: Balance between metric freshness and network load

## Troubleshooting

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
