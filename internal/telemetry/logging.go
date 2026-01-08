package telemetry

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
)

// InitializeLoggerProvider creates and configures the OTEL LoggerProvider
func InitializeLoggerProvider(ctx context.Context, cfg *TelemetryConfig) (*log.LoggerProvider, error) {
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

	var exporter log.Exporter

	if cfg.ExportType == "otlp" {
		// Use OTLP gRPC exporter for logs
		opts := []otlploggrpc.Option{
			otlploggrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlploggrpc.WithDialOption(
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50 * 1024 * 1024)),
			),
		}

		// Check for insecure mode
		if os.Getenv("OTEL_EXPORTER_OTLP_INSECURE") == "true" {
			opts = append(opts, otlploggrpc.WithInsecure())
		}

		exporter, err = otlploggrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create otlp log exporter: %w", err)
		}
	} else {
		// Return no-op provider
		return log.NewLoggerProvider(), nil
	}

	processor := log.NewBatchProcessor(exporter)

	lp := log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(processor),
	)

	// Set global LoggerProvider
	global.SetLoggerProvider(lp)

	return lp, nil
}
