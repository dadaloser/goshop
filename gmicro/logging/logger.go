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

// SetDefault installs the logger used by framework packages.
func SetDefault(logger *slog.Logger) {
	if logger == nil {
		return
	}
	defaultMu.Lock()
	defaultLogger = logger
	defaultMu.Unlock()
}

// Default returns the logger used by framework packages.
func Default() *slog.Logger {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultLogger
}

// Info logs a framework event at info level.
func Info(msg string, attrs ...slog.Attr) {
	InfoContext(contextutil.Root(), msg, attrs...)
}

// InfoContext logs a framework event at info level with ctx.
func InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, slog.LevelInfo, msg, attrs...)
}

// Warn logs a framework event at warn level.
func Warn(msg string, attrs ...slog.Attr) {
	WarnContext(contextutil.Root(), msg, attrs...)
}

// WarnContext logs a framework event at warn level with ctx.
func WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	log(ctx, slog.LevelWarn, msg, attrs...)
}

// Error logs a framework event at error level.
func Error(msg string, attrs ...slog.Attr) {
	ErrorContext(contextutil.Root(), msg, attrs...)
}

// ErrorContext logs a framework event at error level with ctx.
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
