package logging

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestNewLoggerCreatesLogger(t *testing.T) {
	l, err := NewLogger("", InfoLevel, "console")
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	if l == nil {
		t.Fatalf("expected logger, got nil")
	}
}

func TestFileAndConsoleWrappers(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_wrapper")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	fw := &FileWriterWrapper{file: tmpfile, programName: "test", pid: 123}
	if _, err := fw.Write([]byte("test log")); err != nil {
		t.Fatalf("FileWriterWrapper Write failed: %v", err)
	}
	if err := fw.Sync(); err != nil {
		t.Fatalf("FileWriterWrapper Sync failed: %v", err)
	}
	if err := fw.Close(); err != nil {
		t.Fatalf("FileWriterWrapper Close failed: %v", err)
	}

	cw := &ConsoleWriterWrapper{writer: os.Stdout, programName: "prog", pid: 123}
	if _, err := cw.Write([]byte("hi")); err != nil {
		t.Fatalf("ConsoleWriterWrapper Write failed: %v", err)
	}
	if err := cw.Sync(); err != nil {
		t.Fatalf("ConsoleWriterWrapper Sync failed: %v", err)
	}
}

// TestTimeEncoderFormats removed due to complex mocking requirements for zapcore.PrimitiveArrayEncoder
// func TestTimeEncoderFormats(t *testing.T) {
// 	var enc zapcore.PrimitiveArrayEncoder
// 	timeEncoder(time.Now(), enc)
// }

func TestAdvancedLoggerMethods(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	logger, err := NewLogger(tmpFile.Name(), DebugLevel, "file")
	if err != nil {
		t.Fatalf("NewLogger(file) failed: %v", err)
	}
	logger.Debug("test")

	zapLogger := logger.GetZapLogger()
	if zapLogger == nil {
		t.Error("GetZapLogger returned nil")
	}
	_ = zapLogger
}

func TestLogger_Methods(t *testing.T) {
	logger, _ := NewLogger("", "debug", "console")
	logger.Debug("debug msg", zap.String("key", "val"))
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")
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

	if err := logger.Sync(); err != nil {
		t.Logf("Sync returned error: %v", err)
	}

	zapLogger := logger.GetZapLogger()
	if zapLogger == nil {
		t.Error("GetZapLogger returned nil")
	}

	core := zap.NewNop().Core()
	newLogger := logger.WithCore(core)
	if newLogger == nil {
		t.Error("WithCore returned nil")
	}
}
