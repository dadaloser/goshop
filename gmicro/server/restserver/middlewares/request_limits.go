package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// StatusResponder renders a framework-generated HTTP failure.
type StatusResponder func(*gin.Context, int)

// AbortWithStatus is the default responder for framework failures.
func AbortWithStatus(c *gin.Context, status int) {
	c.AbortWithStatus(status)
}

// RequestBodyLimit enforces a hard upper bound for every inbound request body.
func RequestBodyLimit(maxBytes int64, responder StatusResponder) gin.HandlerFunc {
	if responder == nil {
		responder = AbortWithStatus
	}
	return func(c *gin.Context) {
		if c.Request.Body == nil || maxBytes <= 0 {
			c.Next()
			return
		}
		if c.Request.ContentLength > maxBytes {
			responder(c, http.StatusRequestEntityTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// RequestDeadline propagates a cooperative deadline through the request
// context. It deliberately does not detach the handler in another goroutine.
func RequestDeadline(timeout time.Duration, responder StatusResponder) gin.HandlerFunc {
	if responder == nil {
		responder = AbortWithStatus
	}
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		c.Next()
		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			responder(c, http.StatusGatewayTimeout)
		}
	}
}
