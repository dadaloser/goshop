package auth

import (
	"encoding/base64"
	"strings"

	"goshop/gmicro/server/restserver/middlewares"

	"github.com/gin-gonic/gin"
)

// BasicStrategy defines Basic authentication strategy.
type BasicStrategy struct {
	compare          func(username string, password string) bool
	failureResponder FailureResponder
}

var _ middlewares.AuthStrategy = &BasicStrategy{}

// NewBasicStrategy create basic strategy with compare function.
func NewBasicStrategy(compare func(username string, password string) bool, options ...Option) BasicStrategy {
	return BasicStrategy{
		compare:          compare,
		failureResponder: resolveFailureResponder(options),
	}
}

// AuthFunc defines basic strategy as the gin authentication middleware.
func (b BasicStrategy) AuthFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)

		if len(auth) != 2 || auth[0] != "Basic" {
			reject(c, b.failureResponder, ErrInvalidCredentials)
			return
		}

		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		pair := strings.SplitN(string(payload), ":", 2)

		if len(pair) != 2 || !b.compare(pair[0], pair[1]) {
			reject(c, b.failureResponder, ErrInvalidCredentials)
			return
		}

		c.Set(middlewares.UsernameKey, pair[0])

		c.Next()
	}
}
