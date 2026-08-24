package auth

import (
	"strings"

	"goshop/gmicro/server/restserver/middlewares"

	"github.com/gin-gonic/gin"
)

const authHeaderCount = 2

// AutoStrategy defines authentication strategy which can automatically choose between Basic and Bearer
// according `Authorization` header.
type AutoStrategy struct {
	basic            BasicStrategy
	jwt              JWTStrategy
	failureResponder FailureResponder
}

var _ middlewares.AuthStrategy = &AutoStrategy{}

// NewAutoStrategy create auto strategy with basic strategy and jwt strategy.
func NewAutoStrategy(basic BasicStrategy, jwt JWTStrategy, options ...Option) AutoStrategy {
	responder := resolveFailureResponder(options)
	if responder != nil {
		basic.failureResponder = responder
		jwt.failureResponder = responder
	}
	return AutoStrategy{
		basic:            basic,
		jwt:              jwt,
		failureResponder: responder,
	}
}

// AuthFunc defines auto strategy as the gin authentication middleware.
func (a AutoStrategy) AuthFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		operator := middlewares.AuthOperator{}
		authHeader := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)

		if len(authHeader) != authHeaderCount {
			reject(c, a.failureResponder, ErrInvalidAuthorization)
			return
		}

		switch authHeader[0] {
		case "Basic":
			operator.SetStrategy(a.basic)
		case "Bearer":
			operator.SetStrategy(a.jwt)
			// a.JWT.MiddlewareFunc()(c)
		default:
			reject(c, a.failureResponder, ErrInvalidCredentials)
			return
		}

		operator.AuthFunc()(c)

		c.Next()
	}
}
