package middlewares

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"goshop/pkg/log"
	ginlog "goshop/pkg/log/gin"
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
		c.Set(log.KeyRequestID, requestID)
		c.Header(requestIDHeader, requestID)

		started := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "__unmatched__"
		}
		log.InfoC(ginlog.Context(c), "http request completed",
			log.String("http_method", c.Request.Method),
			log.String("http_route", route),
			log.Int("http_status", c.Writer.Status()),
			log.Int("response_bytes", c.Writer.Size()),
			log.Duration("duration", time.Since(started)),
			log.String("client_ip", c.ClientIP()),
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
	if responder == nil {
		responder = AbortWithStatus
	}
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.ErrorC(ginlog.Context(c), "http request panic recovered",
			log.Any("panic", recovered),
			log.ByteString("stack", debug.Stack()),
		)
		responder(c, http.StatusInternalServerError)
	})
}
