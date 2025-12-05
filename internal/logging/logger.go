package logging

import (
	"fmt"
	"io"
	"os"
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

// NewLogger creates a new logger with file and/or console output
// mode can be "file", "console", or "both"
func NewLogger(logFile string, logLevel LogLevel, mode string) (*Logger, error) {
	// Get program name and PID
	programName := os.Args[0]
	pid := os.Getpid()

	// Create atomic level for dynamic changes
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		return nil, fmt.Errorf("invalid log level: %s", logLevel)
	}

	// Create encoder config with custom formatting
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

	// Add file output if requested
	if mode == "file" || mode == "both" {
		if logFile == "" {
			return nil, fmt.Errorf("log file path required when file logging is enabled")
		}

		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		// Wrap file with custom writer to add program name and PID
		fileWriter := zapcore.AddSync(&FileWriterWrapper{
			file:        file,
			programName: programName,
			pid:         pid,
		})
		writers = append(writers, fileWriter)
	}

	// Add console output if requested
	if mode == "console" || mode == "both" {
		consoleWriter := zapcore.AddSync(&ConsoleWriterWrapper{
			writer:      os.Stdout,
			programName: programName,
			pid:         pid,
		})
		writers = append(writers, consoleWriter)
	}

	if len(writers) == 0 {
		return nil, fmt.Errorf("invalid logging mode: %s (must be 'file', 'console', or 'both')", mode)
	}

	// Create multi-writer
	var writer zapcore.WriteSyncer
	if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = zapcore.NewMultiWriteSyncer(writers...)
	}

	// Create core and logger
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writer,
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{
		logger:      logger,
		level:       atomic.Value{},
		programName: programName,
		pid:         pid,
	}, nil
}

// timeEncoder formats time in the format: 2006-01-02 15:04:05.000
func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// FileWriterWrapper adds program name and PID to file logs
type FileWriterWrapper struct {
	file        io.WriteCloser
	programName string
	pid         int
}

func (fw *FileWriterWrapper) Write(p []byte) (n int, err error) {
	// For file, add program name and PID prefix
	prefix := fmt.Sprintf("[%s %d] ", fw.programName, fw.pid)
	prefixedData := append([]byte(prefix), p...)
	return fw.file.Write(prefixedData)
}

func (fw *FileWriterWrapper) Sync() error {
	if f, ok := fw.file.(interface{ Sync() error }); ok {
		return f.Sync()
	}
	return nil
}

func (fw *FileWriterWrapper) Close() error {
	return fw.file.Close()
}

// ConsoleWriterWrapper adds program name and PID to console logs
type ConsoleWriterWrapper struct {
	writer      io.Writer
	programName string
	pid         int
}

func (cw *ConsoleWriterWrapper) Write(p []byte) (n int, err error) {
	// For console, add program name and PID prefix
	prefix := fmt.Sprintf("[%s %d] ", cw.programName, cw.pid)
	prefixedData := append([]byte(prefix), p...)
	return cw.writer.Write(prefixedData)
}

func (cw *ConsoleWriterWrapper) Sync() error {
	return nil
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...zapcore.Field) {
	l.logger.Debug(msg, fields...)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...zapcore.Field) {
	l.logger.Info(msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...zapcore.Field) {
	l.logger.Warn(msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...zapcore.Field) {
	l.logger.Error(msg, fields...)
}

// Crit logs a critical message (uses Fatal level)
func (l *Logger) Crit(msg string, fields ...zapcore.Field) {
	l.logger.Fatal(msg, fields...)
}

// GetZapLogger returns the underlying zap logger for compatibility
func (l *Logger) GetZapLogger() *zap.Logger {
	return l.logger
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.logger.Sync()
}
