package log

import "context"

type contextKey struct {
	name string
}

var (
	requestIDContextKey = contextKey{name: KeyRequestID}
	userIDContextKey    = contextKey{name: KeyUserID}
)

// WithRequestID returns ctx with a request ID value for logging.
func WithRequestID(ctx context.Context, value any) context.Context {
	return context.WithValue(ctx, requestIDContextKey, value)
}

// WithUserID returns ctx with a user ID value for logging.
func WithUserID(ctx context.Context, value any) context.Context {
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
