// Package gin adapts Gin request state for the core log package.
package gin

import (
	"context"

	"goshop/pkg/common/util/contextutil"
	"goshop/pkg/log"

	gingonic "github.com/gin-gonic/gin"
)

type correlationKey struct {
	ginKey string
	with   func(context.Context, any) context.Context
}

var contextKeys = []correlationKey{
	{ginKey: log.KeyRequestID, with: log.WithRequestID},
	{ginKey: log.KeyUserID, with: log.WithUserID},
	{ginKey: "requestID", with: log.WithRequestID},
	{ginKey: "userid", with: log.WithUserID},
	{ginKey: "username", with: log.WithUserID},
}

// Context returns the request context enriched with log correlation values
// stored in the Gin context.
func Context(c *gingonic.Context) context.Context {
	if c == nil || c.Request == nil {
		return contextutil.Root()
	}

	ctx := c.Request.Context()
	for _, key := range contextKeys {
		if value, ok := c.Get(key.ginKey); ok {
			ctx = key.with(ctx, value)
		}
	}
	return ctx
}
