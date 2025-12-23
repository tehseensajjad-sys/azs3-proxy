package telemetry

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLoadTelemetryConfig(t *testing.T) {
	cfg := LoadTelemetryConfig()

	if cfg == nil {
		t.Fatal("LoadTelemetryConfig returned nil")
	}

	if cfg.ServiceName != "azs3-proxy" {
		t.Errorf("unexpected ServiceName: %q", cfg.ServiceName)
	}

	if cfg.ServiceVersion != "1.0.0" {
		t.Errorf("unexpected ServiceVersion: %q", cfg.ServiceVersion)
	}

	if cfg.ExportInterval != 30*time.Second {
		t.Errorf("unexpected ExportInterval: %v", cfg.ExportInterval)
	}
}

func TestLoadTelemetryConfigWithEnv(t *testing.T) {
	_ = os.Setenv("SERVICE_NAME", "test-service")
	_ = os.Setenv("SERVICE_VERSION", "2.0.0")
	_ = os.Setenv("TELEMETRY_EXPORT_INTERVAL", "60s")
	defer func() { _ = os.Unsetenv("SERVICE_NAME") }()
	defer func() { _ = os.Unsetenv("SERVICE_VERSION") }()
	defer func() { _ = os.Unsetenv("TELEMETRY_EXPORT_INTERVAL") }()

	cfg := LoadTelemetryConfig()

	if cfg.ServiceName != "test-service" {
		t.Errorf("expected ServiceName 'test-service', got %q", cfg.ServiceName)
	}

	if cfg.ServiceVersion != "2.0.0" {
		t.Errorf("expected ServiceVersion '2.0.0', got %q", cfg.ServiceVersion)
	}

	if cfg.ExportInterval != 60*time.Second {
		t.Errorf("expected ExportInterval 60s, got %v", cfg.ExportInterval)
	}
}

func TestTelemetryConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *TelemetryConfig
		wantErr bool
	}{
		{
			name: "disabled config",
			cfg: &TelemetryConfig{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTelemetryConfigIsEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  *TelemetryConfig
		want bool
	}{
		{
			name: "disabled",
			cfg: &TelemetryConfig{
				Enabled: false,
			},
			want: false,
		},
		{
			name: "enabled",
			cfg: &TelemetryConfig{
				Enabled: true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.IsEnabled()
			if got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	ctx := context.Background()

	mgr, err := NewManager(ctx)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("NewManager returned nil manager")
	}

	// Should be able to call IsEnabled
	_ = mgr.IsEnabled()

	// Should get config
	cfg := mgr.GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig returned nil")
	}
}

func TestNewManagerWithEnabledTelemetry(t *testing.T) {
	_ = os.Setenv("TELEMETRY_ENABLED", "true")
	_ = os.Setenv("TELEMETRY_EXPORT_TYPE", "noop")
	defer func() {
		_ = os.Unsetenv("TELEMETRY_ENABLED")
		_ = os.Unsetenv("TELEMETRY_EXPORT_TYPE")
	}()

	ctx := context.Background()
	mgr, err := NewManager(ctx)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr.IsEnabled() != true {
		t.Error("expected telemetry to be enabled")
	}

	if mgr.GetMetricsProvider() == nil {
		t.Fatal("GetMetricsProvider returned nil")
	}
}

func TestManagerShutdown(t *testing.T) {
	// Use no-op exporter for this test to avoid connection errors
	_ = os.Setenv("TELEMETRY_EXPORT_TYPE", "noop")
	defer func() { _ = os.Unsetenv("TELEMETRY_EXPORT_TYPE") }()

	ctx := context.Background()

	mgr, err := NewManager(ctx)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Shutdown should not error
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = mgr.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Calling shutdown again should be idempotent
	err = mgr.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Second Shutdown failed: %v", err)
	}
}

func TestRecordS3Request(t *testing.T) {
	ctx := context.Background()

	mgr, err := NewManager(ctx)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// These should not panic
	mgr.RecordS3Request(ctx, "GetObject", true, "")
	mgr.RecordS3Request(ctx, "PutObject", false, "AccessDenied")
	mgr.RecordS3Request(ctx, "DeleteObject", false, "NoSuchKey")
	mgr.RecordS3Request(ctx, "ListBuckets", true, "")
}

func TestRecordCacheMetrics(t *testing.T) {
	ctx := context.Background()

	mgr, err := NewManager(ctx)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// These should not panic
	mgr.RecordCacheHit(ctx, "test-key")
	mgr.RecordCacheMiss(ctx, "test-key")
	mgr.RecordCacheEviction(ctx, "size_limit")
	mgr.RecordCacheEviction(ctx, "expired")
	mgr.RecordCacheExpiration(ctx)
	mgr.RecordCacheOperation(ctx, "put")
	mgr.RecordCacheOperation(ctx, "get")
	mgr.RecordCacheOperation(ctx, "delete")
}

func TestRecordMetricsWhenDisabled(t *testing.T) {
	_ = os.Setenv("TELEMETRY_ENABLED", "false")
	defer func() { _ = os.Unsetenv("TELEMETRY_ENABLED") }()

	ctx := context.Background()

	mgr, err := NewManager(ctx)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// These should not panic or error even when telemetry is disabled
	mgr.RecordS3Request(ctx, "GetObject", true, "")
	mgr.RecordCacheHit(ctx, "key")
	mgr.RecordCacheMiss(ctx, "key")
}

func TestGetOrDefault(t *testing.T) {
	tests := []struct {
		value        string
		defaultValue string
		expected     string
	}{
		{"test", "default", "test"},
		{"", "default", "default"},
		{"", "", ""},
	}

	for _, tt := range tests {
		got := getOrDefault(tt.value, tt.defaultValue)
		if got != tt.expected {
			t.Errorf("getOrDefault(%q, %q) = %q, expected %q", tt.value, tt.defaultValue, got, tt.expected)
		}
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		value        string
		defaultValue bool
		expected     bool
	}{
		{"true", false, true},
		{"false", true, false},
		{"1", false, true},
		{"0", true, false},
		{"", false, false},
		{"", true, true},
		{"invalid", false, false},
	}

	for _, tt := range tests {
		got := parseBool(tt.value, tt.defaultValue)
		if got != tt.expected {
			t.Errorf("parseBool(%q, %v) = %v, expected %v", tt.value, tt.defaultValue, got, tt.expected)
		}
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		value        string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{"30s", 0, 30 * time.Second},
		{"1m", 0, 1 * time.Minute},
		{"", 60 * time.Second, 60 * time.Second},
		{"invalid", 30 * time.Second, 30 * time.Second},
	}

	for _, tt := range tests {
		got := parseDuration(tt.value, tt.defaultValue)
		if got != tt.expected {
			t.Errorf("parseDuration(%q, %v) = %v, expected %v", tt.value, tt.defaultValue, got, tt.expected)
		}
	}
}
