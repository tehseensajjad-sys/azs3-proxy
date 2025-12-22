package telemetry

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestOtelZapCore(t *testing.T) {
	core := NewOtelZapCore(zapcore.DebugLevel)

	// Test With
	newCore := core.With([]zapcore.Field{zap.String("key", "value")})
	if newCore == nil {
		t.Error("With returned nil")
	}

	// Test Check
	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Time:    time.Now(),
		Message: "test message",
	}
	checkedEntry := core.Check(entry, nil)
	if checkedEntry == nil {
		t.Error("Check returned nil for enabled level")
	}

	// Test Check disabled
	// ErrorLevel is higher than InfoLevel. If we set enabler to ErrorLevel, InfoLevel should be disabled.
	coreError := NewOtelZapCore(zapcore.ErrorLevel)
	disabledEntry := zapcore.Entry{
		Level: zapcore.InfoLevel,
	}
	checkedEntryDisabled := coreError.Check(disabledEntry, nil)
	if checkedEntryDisabled != nil {
		t.Error("Check returned non-nil for disabled level")
	}

	// Test Write
	err := core.Write(entry, []zapcore.Field{zap.String("key", "value")})
	if err != nil {
		t.Errorf("Write returned error: %v", err)
	}

	// Test Sync
	err = core.Sync()
	if err != nil {
		t.Errorf("Sync returned error: %v", err)
	}
}

func TestZapLevelToOtelSeverity(t *testing.T) {
	tests := []struct {
		level    zapcore.Level
		expected int // log.Severity is not exported easily as int, but we can check values if we import log
	}{
		{zapcore.DebugLevel, 5},  // SeverityDebug
		{zapcore.InfoLevel, 9},   // SeverityInfo
		{zapcore.WarnLevel, 13},  // SeverityWarn
		{zapcore.ErrorLevel, 17}, // SeverityError
		{zapcore.FatalLevel, 21}, // SeverityFatal
	}

	for _, tt := range tests {
		sev := zapLevelToOtelSeverity(tt.level)
		if int(sev) != tt.expected {
			t.Errorf("zapLevelToOtelSeverity(%v) = %v, want %v", tt.level, sev, tt.expected)
		}
	}
}
