package logging

import (
	"context"
	"log/slog"
	"sync"

	"goshop/gmicro/contextutil"
)

var (
	defaultMu     sync.RWMutex
	defaultLogger = slog.Default()
)

func SetDefault(logger *slog.Logger) {
	if logger == nil {
		return
	}
	defaultMu.Lock()
	defaultLogger = logger
	defaultMu.Unlock()
}

func Default() *slog.Logger {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultLogger
}

func Info(msg string, attrs ...slog.Attr) {
	InfoContext(contextutil.Root(), msg, attrs...)
}

func InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, slog.LevelInfo, msg, attrs...)
}

func Warn(msg string, attrs ...slog.Attr) {
	WarnContext(contextutil.Root(), msg, attrs...)
}

func WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, slog.LevelWarn, msg, attrs...)
}

func Error(msg string, attrs ...slog.Attr) {
	ErrorContext(contextutil.Root(), msg, attrs...)
}

func ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, slog.LevelError, msg, attrs...)
}

func log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	if ctx == nil {
		ctx = contextutil.Root()
	}
	logger := Default()
	if !logger.Enabled(ctx, level) {
		return
	}
	logger.LogAttrs(ctx, level, msg, attrs...)
}
