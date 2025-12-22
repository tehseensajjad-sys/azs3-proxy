package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel represents the logging level
type LogLevel string

const (
	// Log levels
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	CritLevel  LogLevel = "crit"
)

// Logger wraps zap.Logger with dynamic level support
type Logger struct {
	logger      *zap.Logger
	level       atomic.Value // stores *zapcore.AtomicLevel
	programName string
	pid         int
}

// NewLogger creates a new logger with file and/or console output.
// mode can be "file", "console", or "both".
// Returns a Logger configured with the specified level and output destination.
func NewLogger(logFile string, logLevel LogLevel, mode string) (*Logger, error) {
	// Get program name and PID to include in log output
	programName := filepath.Base(os.Args[0])
	pid := os.Getpid()

	// Create atomic level for potential dynamic log level changes in the future
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		return nil, fmt.Errorf("invalid log level: %s", logLevel)
	}

	// Configure JSON encoder with structured fields and custom time format
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    "func",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     timeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var writers []zapcore.WriteSyncer

	// Add file output writer if requested
	if mode == "file" || mode == "both" {
		if logFile == "" {
			return nil, fmt.Errorf("log file path required when file logging is enabled")
		}

		// Open log file for appending, creating if it doesn't exist
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		// Wrap file writer to add program name and PID prefix to each log line
		fileWriter := zapcore.AddSync(&FileWriterWrapper{
			file:        file,
			programName: programName,
			pid:         pid,
		})
		writers = append(writers, fileWriter)
	}

	// Add console output writer if requested
	if mode == "console" || mode == "both" {
		consoleWriter := zapcore.AddSync(&ConsoleWriterWrapper{
			writer:      os.Stdout,
			programName: programName,
			pid:         pid,
		})
		writers = append(writers, consoleWriter)
	}

	// Validate that at least one output writer was configured
	if len(writers) == 0 {
		return nil, fmt.Errorf("invalid logging mode: %s (must be 'file', 'console', or 'both')", mode)
	}

	// Create multi-writer syncer that writes to all configured destinations
	var writer zapcore.WriteSyncer
	if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = zapcore.NewMultiWriteSyncer(writers...)
	}

	// Create core logger with JSON encoder, synced writer, and log level
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writer,
		level,
	)

	// Initialize zap logger with caller tracking and stack traces on errors
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{
		logger:      logger,
		level:       atomic.Value{},
		programName: programName,
		pid:         pid,
	}, nil
}

// timeEncoder formats time in the format: 2006-01-02 15:04:05.000
// timeEncoder formats time in ISO format with millisecond precision: 2006-01-02 15:04:05.000
func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// FileWriterWrapper wraps an output file and adds program name and PID prefix to each log line
type FileWriterWrapper struct {
	file        io.WriteCloser // File handle for writing logs
	programName string         // Name of the binary
	pid         int            // Process ID
}

// Write adds program name and PID prefix before writing to file
func (fw *FileWriterWrapper) Write(p []byte) (n int, err error) {
	prefix := fmt.Sprintf("[%s %d] ", fw.programName, fw.pid)
	prefixedData := append([]byte(prefix), p...)
	return fw.file.Write(prefixedData)
}

// Sync flushes the file buffer to disk if supported
func (fw *FileWriterWrapper) Sync() error {
	if f, ok := fw.file.(interface{ Sync() error }); ok {
		return f.Sync()
	}
	return nil
}

// Close closes the underlying file handle
func (fw *FileWriterWrapper) Close() error {
	return fw.file.Close()
}

// ConsoleWriterWrapper wraps stdout and adds program name and PID prefix to each log line
type ConsoleWriterWrapper struct {
	writer      io.Writer // Standard output writer
	programName string    // Name of the binary
	pid         int       // Process ID
}

// Write adds program name and PID prefix before writing to console
func (cw *ConsoleWriterWrapper) Write(p []byte) (n int, err error) {
	prefix := fmt.Sprintf("[%s %d] ", cw.programName, cw.pid)
	prefixedData := append([]byte(prefix), p...)
	return cw.writer.Write(prefixedData)
}

// Sync is a no-op for console output as it cannot be buffered
func (cw *ConsoleWriterWrapper) Sync() error {
	return nil
}

// Debug logs a message at DEBUG level with optional structured fields
func (l *Logger) Debug(msg string, fields ...zapcore.Field) {
	l.logger.Debug(msg, fields...)
}

// Info logs a message at INFO level with optional structured fields
func (l *Logger) Info(msg string, fields ...zapcore.Field) {
	l.logger.Info(msg, fields...)
}

// Warn logs a message at WARN level with optional structured fields
func (l *Logger) Warn(msg string, fields ...zapcore.Field) {
	l.logger.Warn(msg, fields...)
}

// Error logs a message at ERROR level with optional structured fields
func (l *Logger) Error(msg string, fields ...zapcore.Field) {
	l.logger.Error(msg, fields...)
}

// Crit logs a CRITICAL message and exits the program (uses Fatal level internally)
func (l *Logger) Crit(msg string, fields ...zapcore.Field) {
	l.logger.Fatal(msg, fields...)
}

// GetZapLogger returns the underlying zap.Logger for compatibility with code that expects *zap.Logger
func (l *Logger) GetZapLogger() *zap.Logger {
	return l.logger
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.logger.Sync()
}

// WithCore returns a new Logger with the given core added to the existing logger
func (l *Logger) WithCore(core zapcore.Core) *Logger {
	newLogger := l.logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, core)
	}))

	return &Logger{
		logger:      newLogger,
		level:       l.level,
		programName: l.programName,
		pid:         l.pid,
	}
}
