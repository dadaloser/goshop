package log

import (
	"context"
	"log/slog"
	"slices"

	"goshop/gmicro/logging"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Slog returns a slog logger backed by the configured zap logger.
func Slog() *slog.Logger {
	mu.Lock()
	defer mu.Unlock()
	return newSlogLogger(std)
}

func newSlogLogger(logger *Logger) *slog.Logger {
	return slog.New(&slogHandler{logger: logger})
}

type slogHandler struct {
	logger *Logger
	fields []zapcore.Field
	groups []string
}

func (h *slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h != nil && h.logger != nil && h.logger.Check(slogZapLevel(level), "") != nil
}

func (h *slogHandler) Handle(ctx context.Context, record slog.Record) error {
	if h == nil || h.logger == nil {
		return nil
	}
	fields := make([]zapcore.Field, 0, len(h.fields)+record.NumAttrs())
	fields = append(fields, h.fields...)
	record.Attrs(func(attr slog.Attr) bool {
		fields = appendSlogAttr(fields, h.groups, attr)
		return true
	})
	switch {
	case record.Level >= slog.LevelError:
		h.logger.ErrorContext(ctx, record.Message, fields...)
	case record.Level >= slog.LevelWarn:
		h.logger.WarnContext(ctx, record.Message, fields...)
	case record.Level >= slog.LevelInfo:
		h.logger.InfoContext(ctx, record.Message, fields...)
	default:
		h.logger.DebugContext(ctx, record.Message, fields...)
	}
	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.fields = slices.Clone(h.fields)
	for _, attr := range attrs {
		clone.fields = appendSlogAttr(clone.fields, h.groups, attr)
	}
	return &clone
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.groups = append(slices.Clone(h.groups), name)
	return &clone
}

func appendSlogAttr(fields []zapcore.Field, groups []string, attr slog.Attr) []zapcore.Field {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return fields
	}
	key := attr.Key
	for i := len(groups) - 1; i >= 0; i-- {
		key = groups[i] + "." + key
	}
	return append(fields, zap.Any(key, attr.Value.Any()))
}

func slogZapLevel(level slog.Level) zapcore.Level {
	switch {
	case level >= slog.LevelError:
		return zapcore.ErrorLevel
	case level >= slog.LevelWarn:
		return zapcore.WarnLevel
	case level >= slog.LevelInfo:
		return zapcore.InfoLevel
	default:
		return zapcore.DebugLevel
	}
}

func installFrameworkLogger(logger *Logger) {
	logging.SetDefault(newSlogLogger(logger))
}
