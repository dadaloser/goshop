package logging

import (
	"context"
	"fmt"
	"strconv"

	"goshop/gmicro/contextutil"
)

const (
	// KeyRequestID is the structured log field and context key name for request
	// correlation.
	KeyRequestID = "request_id"
	// KeyUserID is the structured log field and context key name for user
	// correlation.
	KeyUserID = "user_id"
)

type contextKey struct {
	name string
}

var (
	requestIDContextKey = contextKey{name: KeyRequestID}
	userIDContextKey    = contextKey{name: KeyUserID}
)

// WithRequestID returns ctx with a request ID value for logging.
func WithRequestID(ctx context.Context, value any) context.Context {
	if ctx == nil {
		ctx = contextutil.Root()
	}
	return context.WithValue(ctx, requestIDContextKey, value)
}

// WithUserID returns ctx with a user ID value for logging.
func WithUserID(ctx context.Context, value any) context.Context {
	if ctx == nil {
		ctx = contextutil.Root()
	}
	return context.WithValue(ctx, userIDContextKey, value)
}

// RequestIDFromContext returns the logging request ID from ctx.
func RequestIDFromContext(ctx context.Context) string {
	return contextStringValue(ctx, requestIDContextKey, KeyRequestID, "requestID")
}

// UserIDFromContext returns the logging user ID from ctx.
func UserIDFromContext(ctx context.Context) string {
	return contextStringValue(ctx, userIDContextKey, KeyUserID, "userid", "username")
}

func contextStringValue(ctx context.Context, keys ...any) string {
	if ctx == nil {
		return ""
	}
	for _, key := range keys {
		if key == nil {
			continue
		}
		if s, ok := key.(string); ok && s == "" {
			continue
		}
		value := ctx.Value(key)
		switch v := value.(type) {
		case string:
			if v != "" {
				return v
			}
		case fmt.Stringer:
			if s := v.String(); s != "" {
				return s
			}
		case int:
			return strconv.Itoa(v)
		case int8, int16, int32, int64:
			return fmt.Sprintf("%d", v)
		case uint, uint8, uint16, uint32, uint64:
			return fmt.Sprintf("%d", v)
		case float32:
			return strconv.FormatFloat(float64(v), 'f', -1, 32)
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		}
	}
	return ""
}
