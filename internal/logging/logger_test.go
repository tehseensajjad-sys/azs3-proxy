package logging

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	// Test Console Mode
	logger, err := NewLogger("", "info", "console")
	if err != nil {
		t.Fatalf("NewLogger(console) failed: %v", err)
	}
	if logger == nil {
		t.Fatal("NewLogger(console) returned nil")
	}
	logger.Info("test console log")

	// Test File Mode
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	logger, err = NewLogger(tmpFile.Name(), "debug", "file")
	if err != nil {
		t.Fatalf("NewLogger(file) failed: %v", err)
	}
	logger.Debug("test file log")

	// Test Both Mode
	logger, err = NewLogger(tmpFile.Name(), "warn", "both")
	if err != nil {
		t.Fatalf("NewLogger(both) failed: %v", err)
	}
	logger.Warn("test both log")

	// Test Invalid Level
	_, err = NewLogger("", "invalid", "console")
	if err == nil {
		t.Error("Expected error for invalid log level")
	}

	// Test Invalid Mode
	_, err = NewLogger("", "info", "invalid")
	if err == nil {
		t.Error("Expected error for invalid mode")
	}

	// Test Missing File
	_, err = NewLogger("", "info", "file")
	if err == nil {
		t.Error("Expected error for missing log file")
	}
}

func TestLogger_Methods(t *testing.T) {
	logger, _ := NewLogger("", "debug", "console")

	// Verify methods don't panic
	logger.Debug("debug msg", zap.String("key", "val"))
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")
	// logger.Fatal calls os.Exit, so we skip it
}

func TestLogger_AdvancedMethods(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-log-advanced-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	logger, err := NewLogger(tmpFile.Name(), "info", "file")
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	// Test Sync
	if err := logger.Sync(); err != nil {
		// Sync might fail on some systems/files, but we just want to cover the call
		t.Logf("Sync returned error: %v", err)
	}

	// Test GetZapLogger
	zapLogger := logger.GetZapLogger()
	if zapLogger == nil {
		t.Error("GetZapLogger returned nil")
	}

	// Test WithCore
	// Create a dummy core
	core := zap.NewNop().Core()
	newLogger := logger.WithCore(core)
	if newLogger == nil {
		t.Error("WithCore returned nil")
	}
}

func TestFileWriterWrapper(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_wrapper")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	wrapper := &FileWriterWrapper{
		file:        tmpfile,
		programName: "test",
		pid:         123,
	}

	_, err = wrapper.Write([]byte("test log"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	err = wrapper.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	err = wrapper.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
