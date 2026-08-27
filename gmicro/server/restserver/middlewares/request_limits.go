package middlewares

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// StatusResponder renders a framework-generated HTTP failure. The framework
// aborts the request after this function returns.
type StatusResponder func(*gin.Context, int)

// AbortWithStatus is the default responder for framework failures.
func AbortWithStatus(c *gin.Context, status int) {
	c.Status(status)
}

// Respond writes a framework failure and always stops the remaining handlers.
func Respond(c *gin.Context, responder StatusResponder, status int) {
	if responder == nil {
		responder = AbortWithStatus
	}
	responder(c, status)
	c.Abort()
}

// RequestBodyLimit enforces a hard upper bound for every inbound request body.
func RequestBodyLimit(maxBytes int64, responder StatusResponder) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil || maxBytes <= 0 {
			c.Next()
			return
		}
		if c.Request.ContentLength > maxBytes {
			Respond(c, responder, http.StatusRequestEntityTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// RequestDeadline propagates a cooperative deadline through the request
// context. It deliberately does not detach the handler in another goroutine.
func RequestDeadline(timeout time.Duration, responder StatusResponder) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		c.Next()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) && !c.Writer.Written() {
			Respond(c, responder, http.StatusGatewayTimeout)
		}
	}
}
