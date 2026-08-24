// Package gin adapts Gin request state for the core log package.
package gin

import (
	"context"

	"goshop/pkg/log"

	gingonic "github.com/gin-gonic/gin"
)

var contextKeys = []string{
	log.KeyRequestID,
	log.KeyUserID,
	"requestID",
	"userid",
	"username",
}

// Context returns the request context enriched with log correlation values
// stored in the Gin context.
func Context(c *gingonic.Context) context.Context {
	if c == nil || c.Request == nil {
		return context.Background()
	}

	ctx := c.Request.Context()
	for _, key := range contextKeys {
		if value, ok := c.Get(key); ok {
			ctx = context.WithValue(ctx, key, value)
		}
	}
	return ctx
}
