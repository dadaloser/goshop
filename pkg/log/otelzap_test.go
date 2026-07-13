package log

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type Test struct {
	log     func(ctx context.Context, log *Logger)
	require func(t *testing.T, event sdktrace.Event)
}

func TestOtelZap(t *testing.T) {
	tests := []Test{
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).Info("hello")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "INFO", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.InfoContext(ctx, "hello")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "INFO", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).Warn("hello", zap.String("foo", "bar"))
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "WARN", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.Equal(t, "bar", foo.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).Warn("hello", zap.String("password", "secret-123"))
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				password, ok := m["password"]
				require.True(t, ok)
				require.Equal(t, redactedFieldValue, password.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).Warn("hello", zap.Strings("foo", []string{"bar1", "bar2", "bar3"}))
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "WARN", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.Equal(t, []string{"bar1", "bar2", "bar3"}, foo.AsStringSlice())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).
					WithOptions(zap.Fields(zap.String("baz", "baz1"))).
					WithOptions(zap.Fields(zap.String("faz", "faz1"))).
					Warn("hello", zap.Strings("foo", []string{"bar1", "bar2", "bar3"}))
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "WARN", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.Equal(t, []string{"bar1", "bar2", "bar3"}, foo.AsStringSlice())

				baz, ok := m["baz"]
				require.True(t, ok)
				require.Equal(t, "baz1", baz.AsString())

				faz, ok := m["faz"]
				require.True(t, ok)
				require.Equal(t, "faz1", faz.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).Warn("hello", zap.Durations("foo", []time.Duration{time.Millisecond, time.Second, time.Hour}))
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "WARN", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.Equal(t, []string{"1ms", "1s", "1h0m0s"}, foo.AsStringSlice())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				err := errors.New("some error")
				log.Ctx(ctx).Error("hello", zap.Error(err))
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				excTyp, ok := m[semconv.ExceptionTypeKey]
				require.True(t, ok)
				require.Equal(t, "*errors.errorString", excTyp.AsString())

				excMsg, ok := m[semconv.ExceptionMessageKey]
				require.True(t, ok)
				require.Equal(t, "some error", excMsg.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log = log.Clone(WithStackTrace(true))
				log.Ctx(ctx).Info("hello")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				stack, ok := m[semconv.ExceptionStacktraceKey]
				require.True(t, ok)
				require.NotZero(t, stack.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().ErrorwContext(ctx, "hello", "foo", "bar")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.NotZero(t, foo.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().ErrorwContext(ctx, "hello", "authorization", "Bearer secret")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				auth, ok := m["authorization"]
				require.True(t, ok)
				require.Equal(t, redactedFieldValue, auth.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().ErrorfContext(ctx, "hello %s", "world")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello world", msg.AsString())

				tpl, ok := m[logTemplateKey]
				require.True(t, ok)
				require.Equal(t, "hello %s", tpl.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().Ctx(ctx).Errorw("hello", "foo", "bar")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.NotZero(t, foo.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().Ctx(ctx).Errorf("hello %s", "world")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello world", msg.AsString())

				tpl, ok := m[logTemplateKey]
				require.True(t, ok)
				require.Equal(t, "hello %s", tpl.AsString())

				requireCodeAttrs(t, m)
			},
		},
	}

	logger := New(NewOptions())

	for i, test := range tests {
		test := test
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer("test")

			ctx := context.Background()
			ctx, span := tracer.Start(ctx, "main")

			test.log(ctx, logger)

			span.End()

			spans := sr.Ended()
			require.Equal(t, 1, len(spans))

			events := spans[0].Events()
			require.Equal(t, 1, len(events))

			event := events[0]
			require.Equal(t, "log", event.Name)
			test.require(t, event)
		})
	}
}

func TestLoggerFormattedContextLevels(t *testing.T) {
	tests := []struct {
		name   string
		level  zapcore.Level
		panics bool
		log    func(*Logger)
	}{
		{
			name:  "warn",
			level: zap.WarnLevel,
			log: func(logger *Logger) {
				logger.WarnfContext(context.Background(), "hello %s", "world")
			},
		},
		{
			name:  "error",
			level: zap.ErrorLevel,
			log: func(logger *Logger) {
				logger.ErrorfContext(context.Background(), "hello %s", "world")
			},
		},
		{
			name:  "dpanic",
			level: zap.DPanicLevel,
			log: func(logger *Logger) {
				logger.DPanicfContext(context.Background(), "hello %s", "world")
			},
		},
		{
			name:   "panic",
			level:  zap.PanicLevel,
			panics: true,
			log: func(logger *Logger) {
				logger.PanicfContext(context.Background(), "hello %s", "world")
			},
		},
		{
			name:   "fatal",
			level:  zap.FatalLevel,
			panics: true,
			log: func(logger *Logger) {
				logger.FatalfContext(context.Background(), "hello %s", "world")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			base := zap.New(core, zap.WithFatalHook(zapcore.WriteThenPanic))
			logger := &Logger{
				Logger:           base,
				skipCaller:       base,
				minLevel:         zap.DebugLevel,
				errorStatusLevel: zap.ErrorLevel,
			}

			if tt.panics {
				require.Panics(t, func() { tt.log(logger) })
			} else {
				require.NotPanics(t, func() { tt.log(logger) })
			}

			entries := logs.All()
			require.Len(t, entries, 1)
			require.Equal(t, tt.level, entries[0].Level)
			require.Equal(t, "hello world", entries[0].Message)
		})
	}
}

func TestLoggerGinContextFields(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		username  string
		want      map[string]interface{}
	}{
		{
			name:      "non-empty values are logged",
			requestID: "req-123",
			username:  "alice",
			want: map[string]interface{}{
				KeyRequestID: "req-123",
				KeyUsername:  "alice",
			},
		},
		{
			name: "empty values are omitted",
			want: map[string]interface{}{},
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.InfoLevel)
			base := zap.New(core)
			logger := &Logger{
				Logger:           base,
				skipCaller:       base,
				minLevel:         zap.InfoLevel,
				errorStatusLevel: zap.ErrorLevel,
			}

			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("GET", "/", nil)
			ctx.Set(KeyRequestID, tt.requestID)
			ctx.Set(KeyUsername, tt.username)

			logger.InfoContext(ctx, "hello")

			entries := logs.All()
			require.Len(t, entries, 1)
			fields := entries[0].ContextMap()
			for _, key := range []string{KeyRequestID, KeyUsername} {
				value, ok := tt.want[key]
				if !ok {
					require.NotContains(t, fields, key)
					continue
				}
				require.Equal(t, value, fields[key])
			}
		})
	}
}

func TestFatalCUsesFatalLevel(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	base := zap.New(core, zap.WithFatalHook(zapcore.WriteThenPanic))
	logger := &Logger{
		Logger:           base,
		skipCaller:       base,
		minLevel:         zap.DebugLevel,
		errorStatusLevel: zap.ErrorLevel,
	}

	previous := std
	std = logger
	t.Cleanup(func() { std = previous })

	require.Panics(t, func() {
		FatalC(context.Background(), "fatal")
	})
	entries := logs.All()
	require.Len(t, entries, 1)
	require.Equal(t, zap.FatalLevel, entries[0].Level)
}

func TestLoggerMasksSensitiveFieldsBeforeWrite(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	base := zap.New(wrapSensitiveFieldCore(core))

	base.Info("hello",
		zap.String("password", "secret-123"),
		zap.String("username", "alice"),
	)
	base.Sugar().Infow("hello", "authorization", "Bearer secret", "request_id", "req-1")

	entries := logs.All()
	require.Len(t, entries, 2)
	require.Equal(t, redactedFieldValue, entries[0].ContextMap()["password"])
	require.Equal(t, "alice", entries[0].ContextMap()["username"])
	require.Equal(t, redactedFieldValue, entries[1].ContextMap()["authorization"])
	require.Equal(t, "req-1", entries[1].ContextMap()["request_id"])
}

func requireCodeAttrs(t *testing.T, m map[attribute.Key]attribute.Value) {
	fn, ok := m[semconv.CodeFunctionKey]
	require.True(t, ok)
	require.Contains(t, fn.AsString(), "TestOtelZap")

	file, ok := m[semconv.CodeFilepathKey]
	require.True(t, ok)
	require.Contains(t, file.AsString(), "log/otelzap_test.go")

	_, ok = m[semconv.CodeLineNumberKey]
	require.True(t, ok)
}

func attrMap(attrs []attribute.KeyValue) map[attribute.Key]attribute.Value {
	m := make(map[attribute.Key]attribute.Value, len(attrs))
	for _, kv := range attrs {
		m[kv.Key] = kv.Value
	}
	return m
}
