package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.uber.org/zap/zapcore"
)

// OtelZapCore is a zapcore.Core that sends logs to OpenTelemetry
type OtelZapCore struct {
	zapcore.LevelEnabler
	logger log.Logger
	fields []zapcore.Field
}

// NewOtelZapCore creates a new OtelZapCore
func NewOtelZapCore(enabler zapcore.LevelEnabler) *OtelZapCore {
	return &OtelZapCore{
		LevelEnabler: enabler,
		logger:       global.Logger("github.com/vibhansa-msft/azs3-proxy"),
	}
}

func (c *OtelZapCore) With(fields []zapcore.Field) zapcore.Core {
	return &OtelZapCore{
		LevelEnabler: c.LevelEnabler,
		logger:       c.logger,
		fields:       append(c.fields, fields...),
	}
}

func (c *OtelZapCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *OtelZapCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	r := log.Record{}
	r.SetTimestamp(ent.Time)
	r.SetSeverity(zapLevelToOtelSeverity(ent.Level))
	r.SetSeverityText(ent.Level.String())
	r.SetBody(log.StringValue(ent.Message))

	// Convert zap fields to OTel attributes
	var attrs []log.KeyValue

	// Add fields from the core
	for _, f := range c.fields {
		attrs = append(attrs, zapFieldToOtelAttr(f))
	}

	// Add fields from the entry
	for _, f := range fields {
		attrs = append(attrs, zapFieldToOtelAttr(f))
	}

	r.AddAttributes(attrs...)

	c.logger.Emit(context.Background(), r)

	return nil
}

func zapFieldToOtelAttr(f zapcore.Field) log.KeyValue {
	switch f.Type {
	case zapcore.StringType:
		return log.String(f.Key, f.String)
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return log.Int64(f.Key, f.Integer)
	case zapcore.BoolType:
		return log.Bool(f.Key, f.Integer == 1)
	case zapcore.ErrorType:
		if err, ok := f.Interface.(error); ok {
			return log.String(f.Key, err.Error())
		}
		return log.String(f.Key, "<error>")
	default:
		// Fallback for other types
		return log.String(f.Key, "unsupported_type")
	}
}

func (c *OtelZapCore) Sync() error {
	return nil
}

func zapLevelToOtelSeverity(l zapcore.Level) log.Severity {
	switch l {
	case zapcore.DebugLevel:
		return log.SeverityDebug
	case zapcore.InfoLevel:
		return log.SeverityInfo
	case zapcore.WarnLevel:
		return log.SeverityWarn
	case zapcore.ErrorLevel:
		return log.SeverityError
	case zapcore.DPanicLevel:
		return log.SeverityFatal
	case zapcore.PanicLevel:
		return log.SeverityFatal
	case zapcore.FatalLevel:
		return log.SeverityFatal
	default:
		return log.SeverityInfo
	}
}
