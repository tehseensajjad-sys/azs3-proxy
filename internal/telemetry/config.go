package telemetry

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// TelemetryConfig holds OpenTelemetry configuration
type TelemetryConfig struct {
	Enabled                bool
	ServiceName            string
	ServiceVersion         string
	Environment            string
	ExportType             string
	ExportInterval         time.Duration
	AzureMonitorEnabled    bool
	AzureMonitorConnString string
	OTLPEnabled            bool
	OTLPEndpoint           string
	PrometheusEnabled      bool
	PrometheusPort         int
	PrometheusPath         string
	LogLevel               string
	LogExportEnabled       bool
	MetricsExportEnabled   bool
	BatchSize              int
	ExportTimeout          time.Duration
	MaxQueueSize           int
}

// LoadTelemetryConfig loads configuration from environment
func LoadTelemetryConfig() *TelemetryConfig {
	return &TelemetryConfig{
		Enabled:                parseBool(os.Getenv("TELEMETRY_ENABLED"), true),
		ServiceName:            getOrDefault(os.Getenv("SERVICE_NAME"), "s3-azure-proxy"),
		ServiceVersion:         getOrDefault(os.Getenv("SERVICE_VERSION"), "1.0.0"),
		Environment:            getOrDefault(os.Getenv("ENVIRONMENT"), "development"),
		ExportType:             getOrDefault(os.Getenv("TELEMETRY_EXPORT_TYPE"), "azuremonitor"),
		ExportInterval:         parseDuration(os.Getenv("TELEMETRY_EXPORT_INTERVAL"), 30*time.Second),
		AzureMonitorEnabled:    parseBool(os.Getenv("AZURE_MONITOR_ENABLED"), false),
		AzureMonitorConnString: os.Getenv("AZURE_MONITOR_CONNECTION_STRING"),
		OTLPEnabled:            parseBool(os.Getenv("OTLP_ENABLED"), false),
		OTLPEndpoint:           getOrDefault(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), "http://localhost:4318"),
		PrometheusEnabled:      parseBool(os.Getenv("PROMETHEUS_ENABLED"), false),
		PrometheusPort:         parseInt(os.Getenv("PROMETHEUS_PORT"), 8888),
		PrometheusPath:         getOrDefault(os.Getenv("PROMETHEUS_PATH"), "/metrics"),
		LogLevel:               getOrDefault(os.Getenv("TELEMETRY_LOG_LEVEL"), "info"),
		LogExportEnabled:       parseBool(os.Getenv("TELEMETRY_LOGS_ENABLED"), false),
		MetricsExportEnabled:   parseBool(os.Getenv("TELEMETRY_METRICS_ENABLED"), false),
		BatchSize:              parseInt(os.Getenv("TELEMETRY_BATCH_SIZE"), 512),
		ExportTimeout:          parseDuration(os.Getenv("TELEMETRY_EXPORT_TIMEOUT"), 30*time.Second),
		MaxQueueSize:           parseInt(os.Getenv("TELEMETRY_MAX_QUEUE_SIZE"), 2048),
	}
}

// Validate checks the configuration
func (c *TelemetryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.MetricsExportEnabled && c.AzureMonitorEnabled && c.AzureMonitorConnString == "" {
		return fmt.Errorf("azure monitor enabled but no connection string provided")
	}

	if c.PrometheusEnabled && (c.PrometheusPort < 1 || c.PrometheusPort > 65535) {
		return fmt.Errorf("prometheus port must be between 1 and 65535")
	}

	return nil
}

// IsEnabled checks if telemetry is enabled
func (c *TelemetryConfig) IsEnabled() bool {
	return c.Enabled && (c.MetricsExportEnabled || c.LogExportEnabled)
}

// Helper functions
func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func parseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return b
}

func parseInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return i
}

func parseDuration(value string, defaultValue time.Duration) time.Duration {
	if value == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return d
}
