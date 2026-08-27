package middlewares

import (
	"context"
	"goshop/gmicro/contextutil"
	"goshop/gmicro/logging"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

// RequestLogger emits one structured access log and provides a request ID for
// correlation when the caller did not supply one.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isLowNoiseManagementPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" || len(requestID) > 128 {
			requestID = uuid.NewString()
		}
		c.Set(logging.KeyRequestID, requestID)
		c.Header(requestIDHeader, requestID)

		started := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "__unmatched__"
		}
		logging.InfoContext(ginContext(c), "http request completed",
			slog.String("http_method", c.Request.Method),
			slog.String("http_route", route),
			slog.Int("http_status", c.Writer.Status()),
			slog.Int("response_bytes", c.Writer.Size()),
			slog.Duration("duration", time.Since(started)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}

func isLowNoiseManagementPath(path string) bool {
	switch path {
	case "/metrics", "/livez", "/readyz", "/healthz":
		return true
	default:
		return false
	}
}

// Recovery converts panics into a generic 500 response and records structured
// diagnostics through the application logger.
func Recovery(responder StatusResponder) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logging.ErrorContext(ginContext(c), "http request panic recovered",
			slog.Any("panic", recovered),
			slog.String("stack", string(debug.Stack())),
		)
		Respond(c, responder, http.StatusInternalServerError)
	})
}

func ginContext(c *gin.Context) context.Context {
	if c == nil || c.Request == nil {
		return contextutil.Root()
	}
	ctx := c.Request.Context()
	for _, key := range []string{logging.KeyRequestID, "requestID"} {
		if value, ok := c.Get(key); ok {
			ctx = logging.WithRequestID(ctx, value)
		}
	}
	for _, key := range []string{logging.KeyUserID, "userid", "username"} {
		if value, ok := c.Get(key); ok {
			ctx = logging.WithUserID(ctx, value)
		}
	}
	return ctx
}
