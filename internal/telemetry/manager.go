package telemetry

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Manager manages the lifecycle of telemetry components
type Manager struct {
	cfg             *TelemetryConfig
	meterProvider   metric.MeterProvider
	tracerProvider  *sdktrace.TracerProvider
	loggerProvider  *sdklog.LoggerProvider
	metricsProvider *MetricsProvider
	shutdownOnce    sync.Once
	shutdownErr     error
}

// NewManager creates and initializes a new telemetry manager
func NewManager(ctx context.Context) (*Manager, error) {
	cfg := LoadTelemetryConfig()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid telemetry config: %w", err)
	}

	if !cfg.IsEnabled() {
		return &Manager{
			cfg: cfg,
		}, nil
	}

	// Initialize meter provider
	meterProvider, err := InitializeMeterProvider(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	// Initialize tracer provider
	var tracerProvider *sdktrace.TracerProvider
	var loggerProvider *sdklog.LoggerProvider
	if cfg.Enabled {
		tracerProvider, err = InitializeTracerProvider(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
		}

		loggerProvider, err = InitializeLoggerProvider(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize logger provider: %w", err)
		}
	}

	// Initialize metrics provider
	metricsProvider, err := NewMetricsProvider(ctx, meterProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics provider: %w", err)
	}

	manager := &Manager{
		cfg:             cfg,
		meterProvider:   meterProvider,
		tracerProvider:  tracerProvider,
		loggerProvider:  loggerProvider,
		metricsProvider: metricsProvider,
	}

	return manager, nil
}

// GetMetricsProvider returns the metrics provider
func (m *Manager) GetMetricsProvider() *MetricsProvider {
	return m.metricsProvider
}

// GetConfig returns the telemetry configuration
func (m *Manager) GetConfig() *TelemetryConfig {
	return m.cfg
}

// IsEnabled checks if telemetry is enabled
func (m *Manager) IsEnabled() bool {
	return m.cfg != nil && m.cfg.IsEnabled()
}

// Shutdown gracefully shuts down the telemetry manager
func (m *Manager) Shutdown(ctx context.Context) error {
	var err error
	m.shutdownOnce.Do(func() {
		if m.meterProvider == nil {
			return
		}

		if sdkProvider, ok := m.meterProvider.(interface{ Shutdown(context.Context) error }); ok {
			err = sdkProvider.Shutdown(ctx)
		}

		if m.tracerProvider != nil {
			if e := m.tracerProvider.Shutdown(ctx); e != nil {
				if err == nil {
					err = e
				} else {
					err = fmt.Errorf("meter shutdown: %v; tracer shutdown: %v", err, e)
				}
			}
		}

		if m.loggerProvider != nil {
			if e := m.loggerProvider.Shutdown(ctx); e != nil {
				if err == nil {
					err = e
				} else {
					err = fmt.Errorf("previous shutdown error: %v; logger shutdown: %v", err, e)
				}
			}
		}
	})
	m.shutdownErr = err
	return err
}
