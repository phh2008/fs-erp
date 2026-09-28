package logger

import (
	"context"
	"log/slog"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapHandler 将 log/slog 记录桥接到 zap 核心
type zapHandler struct {
	zapLogger *zap.Logger
	level     slog.Level
	attrs     []zap.Field
	prefix    string // WithGroup 前缀，如 "req.user_id"
}

func newZapHandler(zapLogger *zap.Logger, level slog.Level) *zapHandler {
	return &zapHandler{zapLogger: zapLogger, level: level}
}

func (h *zapHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level && h.zapLogger.Core().Enabled(zapLevelOf(l))
}

func (h *zapHandler) Handle(_ context.Context, r slog.Record) error {
	fields := make([]zap.Field, 0, len(h.attrs)+r.NumAttrs())
	fields = append(fields, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		fields = append(fields, attrToZap(h.prefix, a)...)
		return true
	})

	if ce := h.zapLogger.Check(zapLevelOf(r.Level), r.Message); ce != nil {
		ce.Time = r.Time
		ce.Write(fields...)
	}
	return nil
}

func (h *zapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]zap.Field{}, h.attrs...), attrsToZap(h.prefix, attrs)...)
	return &clone
}

func (h *zapHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	if clone.prefix == "" {
		clone.prefix = name
	} else {
		clone.prefix = clone.prefix + "." + name
	}
	return &clone
}

func attrsToZap(prefix string, attrs []slog.Attr) []zap.Field {
	fields := make([]zap.Field, 0, len(attrs))
	for _, a := range attrs {
		fields = append(fields, attrToZap(prefix, a)...)
	}
	return fields
}

func attrToZap(prefix string, a slog.Attr) []zap.Field {
	key := a.Key
	if prefix != "" {
		key = prefix + "." + key
	}
	v := a.Value.Resolve()
	if v.Kind() == slog.KindGroup {
		return attrsToZap(key, v.Group())
	}
	switch v.Kind() {
	case slog.KindBool:
		return []zap.Field{zap.Bool(key, v.Bool())}
	case slog.KindInt64:
		return []zap.Field{zap.Int64(key, v.Int64())}
	case slog.KindFloat64:
		return []zap.Field{zap.Float64(key, v.Float64())}
	case slog.KindString:
		return []zap.Field{zap.String(key, v.String())}
	case slog.KindTime:
		return []zap.Field{zap.Time(key, v.Time())}
	case slog.KindDuration:
		return []zap.Field{zap.Duration(key, v.Duration())}
	default:
		return []zap.Field{zap.Any(key, v.Any())}
	}
}

func zapLevelOf(l slog.Level) zapcore.Level {
	switch {
	case l < slog.LevelInfo:
		return zapcore.DebugLevel
	case l < slog.LevelWarn:
		return zapcore.InfoLevel
	case l < slog.LevelError:
		return zapcore.WarnLevel
	default:
		return zapcore.ErrorLevel
	}
}
