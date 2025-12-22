package telemetry

import (
	"os"
	"strconv"
	"time"
)

// TelemetryConfig holds OpenTelemetry configuration
type TelemetryConfig struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	ExportType     string
	ExportInterval time.Duration
	OTLPEndpoint   string
	LogLevel       string
	BatchSize      int
	ExportTimeout  time.Duration
	MaxQueueSize   int
}

// LoadTelemetryConfig loads configuration from environment
func LoadTelemetryConfig() *TelemetryConfig {
	return &TelemetryConfig{
		Enabled:        parseBool(os.Getenv("TELEMETRY_ENABLED"), true),
		ServiceName:    getOrDefault(os.Getenv("SERVICE_NAME"), "azs3-proxy"),
		ServiceVersion: getOrDefault(os.Getenv("SERVICE_VERSION"), "1.0.0"),
		Environment:    getOrDefault(os.Getenv("ENVIRONMENT"), "development"),
		ExportType:     getOrDefault(os.Getenv("TELEMETRY_EXPORT_TYPE"), "otlp"),
		ExportInterval: parseDuration(os.Getenv("TELEMETRY_EXPORT_INTERVAL"), 30*time.Second),
		OTLPEndpoint:   getOrDefault(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), "localhost:4317"),
		LogLevel:       getOrDefault(os.Getenv("TELEMETRY_LOG_LEVEL"), "info"),
		BatchSize:      parseInt(os.Getenv("TELEMETRY_BATCH_SIZE"), 512),
		ExportTimeout:  parseDuration(os.Getenv("TELEMETRY_EXPORT_TIMEOUT"), 30*time.Second),
		MaxQueueSize:   parseInt(os.Getenv("TELEMETRY_MAX_QUEUE_SIZE"), 2048),
	}
}

// Validate checks the configuration
func (c *TelemetryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	return nil
}

// IsEnabled checks if telemetry is enabled
func (c *TelemetryConfig) IsEnabled() bool {
	return c.Enabled
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
