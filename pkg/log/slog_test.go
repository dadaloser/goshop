package log

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestSlogHandlerRespectsLevelAndGroupBinding(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	base := zap.New(core)
	logger := &Logger{
		Logger:           base,
		skipCaller:       base,
		minLevel:         zap.InfoLevel,
		errorStatusLevel: zap.ErrorLevel,
	}

	slogger := newSlogLogger(logger).
		WithGroup("outer").
		With(slog.String("bound", "value")).
		WithGroup("inner")

	slogger.DebugContext(context.Background(), "debug skipped", slog.String("debug", "value"))
	slogger.InfoContext(context.Background(), "info", slog.String("runtime", "value"))

	entries := logs.All()
	require.Len(t, entries, 1)
	require.Equal(t, zapcore.InfoLevel, entries[0].Level)

	fields := entries[0].ContextMap()
	require.Equal(t, "value", fields["outer.bound"])
	require.Equal(t, "value", fields["outer.inner.runtime"])
	require.NotContains(t, fields, "outer.inner.bound")
	require.NotContains(t, fields, "outer.inner.debug")
}
