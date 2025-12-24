package telemetry

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	otlpmetric "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
)

// MetricsProvider holds all metric instruments
type MetricsProvider struct {
	meter metric.Meter

	// S3 API Metrics
	S3RequestsTotal   metric.Int64Counter
	S3RequestsSuccess metric.Int64Counter
	S3RequestsErrors  metric.Int64Counter
	S3ErrorsByType    metric.Int64Counter

	// Cache Metrics
	CacheHitsTotal        metric.Int64Counter
	CacheMissesTotal      metric.Int64Counter
	CacheEvictionsTotal   metric.Int64Counter
	CacheExpirationsTotal metric.Int64Counter
	CacheOperationsTotal  metric.Int64Counter

	// Azure Backend Metrics
	AzureRequestsTotal  metric.Int64Counter
	AzureRequestsErrors metric.Int64Counter
}

// NewMetricsProvider creates all metric instruments
func NewMetricsProvider(ctx context.Context, meterProvider metric.MeterProvider) (*MetricsProvider, error) {
	meter := meterProvider.Meter(
		"github.com/vibhansa-msft/azs3-proxy",
		metric.WithInstrumentationVersion("1.0.0"),
	)

	mp := &MetricsProvider{
		meter: meter,
	}

	var err error

	// S3 API Metrics
	mp.S3RequestsTotal, err = meter.Int64Counter(
		"s3_requests_total",
		metric.WithDescription("Total S3 API requests"),
		metric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3RequestsTotal: %w", err)
	}

	mp.S3RequestsSuccess, err = meter.Int64Counter(
		"s3_requests_success_total",
		metric.WithDescription("Successful S3 API requests"),
		metric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3RequestsSuccess: %w", err)
	}

	mp.S3RequestsErrors, err = meter.Int64Counter(
		"s3_requests_errors_total",
		metric.WithDescription("Failed S3 API requests"),
		metric.WithUnit("{errors}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3RequestsErrors: %w", err)
	}

	mp.S3ErrorsByType, err = meter.Int64Counter(
		"s3_errors_by_type_total",
		metric.WithDescription("S3 errors by type"),
		metric.WithUnit("{errors}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3ErrorsByType: %w", err)
	}

	// Cache Metrics
	mp.CacheHitsTotal, err = meter.Int64Counter(
		"cache_hits_total",
		metric.WithDescription("Cache hits"),
		metric.WithUnit("{hits}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CacheHitsTotal: %w", err)
	}

	mp.CacheMissesTotal, err = meter.Int64Counter(
		"cache_misses_total",
		metric.WithDescription("Cache misses"),
		metric.WithUnit("{misses}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CacheMissesTotal: %w", err)
	}

	mp.CacheEvictionsTotal, err = meter.Int64Counter(
		"cache_evictions_total",
		metric.WithDescription("Cache evictions"),
		metric.WithUnit("{evictions}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CacheEvictionsTotal: %w", err)
	}

	mp.CacheExpirationsTotal, err = meter.Int64Counter(
		"cache_expirations_total",
		metric.WithDescription("Cache expirations (TTL)"),
		metric.WithUnit("{expirations}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CacheExpirationsTotal: %w", err)
	}

	mp.CacheOperationsTotal, err = meter.Int64Counter(
		"cache_operations_total",
		metric.WithDescription("Cache operations"),
		metric.WithUnit("{operations}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CacheOperationsTotal: %w", err)
	}

	// Azure Backend Metrics
	mp.AzureRequestsTotal, err = meter.Int64Counter(
		"backend_requests_total",
		metric.WithDescription("Total Backend Storage requests"),
		metric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create AzureRequestsTotal: %w", err)
	}

	mp.AzureRequestsErrors, err = meter.Int64Counter(
		"backend_requests_errors_total",
		metric.WithDescription("Failed Backend Storage requests"),
		metric.WithUnit("{errors}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create AzureRequestsErrors: %w", err)
	}

	return mp, nil
}

// InitializeMeterProvider creates and configures the OTEL MeterProvider
func InitializeMeterProvider(ctx context.Context, cfg *TelemetryConfig) (metric.MeterProvider, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	if cfg.ExportType == "otlp" {
		opts := []otlpmetric.Option{
			otlpmetric.WithEndpoint(cfg.OTLPEndpoint),
		}

		if os.Getenv("OTEL_EXPORTER_OTLP_INSECURE") == "true" {
			opts = append(opts, otlpmetric.WithInsecure())
		}

		exporter, err := otlpmetric.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create otlp metric exporter: %w", err)
		}
		reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(cfg.ExportInterval))
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(reader))
		return provider, nil
	}

	// Default: no-op exporter via periodic reader
	reader := sdkmetric.NewPeriodicReader(&noOpExporter{}, sdkmetric.WithInterval(cfg.ExportInterval))
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	return provider, nil
}

// noOpExporter is a placeholder exporter
type noOpExporter struct{}

func (e *noOpExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (e *noOpExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return nil
}

func (e *noOpExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	return nil
}

func (e *noOpExporter) ForceFlush(ctx context.Context) error {
	return nil
}

func (e *noOpExporter) Shutdown(ctx context.Context) error {
	return nil
}
