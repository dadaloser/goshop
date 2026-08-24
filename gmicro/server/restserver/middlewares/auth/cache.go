package auth

import (
	stdErrors "errors"
	"strings"
	"time"

	"goshop/gmicro/server/restserver/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Defined errors.
var (
	ErrMissingKID    = stdErrors.New("invalid token format: missing kid field in claims")
	ErrMissingSecret = stdErrors.New("cannot obtain secret information from cache")
)

// Secret contains the basic information of the secret key.
type Secret struct {
	Username string
	ID       string
	Key      string
	Expires  int64
}

// CacheStrategy defines jwt bearer authentication strategy which called `cache strategy`.
// Secrets are obtained through grpc api interface and cached in memory.
type CacheStrategy struct {
	get              func(kid string) (Secret, error)
	failureResponder FailureResponder
}

var _ middlewares.AuthStrategy = &CacheStrategy{}

// NewCacheStrategy create cache strategy with function which can list and cache secrets.
func NewCacheStrategy(get func(kid string) (Secret, error), options ...Option) CacheStrategy {
	return CacheStrategy{get: get, failureResponder: resolveFailureResponder(options)}
}

// AuthFunc defines cache strategy as the gin authentication middleware.
func (cache CacheStrategy) AuthFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Request.Header.Get("Authorization")
		if len(header) == 0 {
			reject(c, cache.failureResponder, ErrMissingCredentials)
			return
		}

		rawJWT, ok := strings.CutPrefix(strings.TrimSpace(header), "Bearer ")
		if !ok || strings.TrimSpace(rawJWT) == "" {
			reject(c, cache.failureResponder, ErrInvalidAuthorization)
			return
		}

		// Use own validation logic, see below
		var secret Secret

		claims := jwt.MapClaims{}
		// Verify the token
		parsedT, err := jwt.ParseWithClaims(rawJWT, claims, func(token *jwt.Token) (interface{}, error) {
			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, ErrMissingKID
			}

			//我们的jwt的以前的认证方式是， 只要解密成功，就认为是合法的
			//如果我有个恶意的用户，他可以伪造一个jwt，然后把kid设置成一个不存在的kid，这样就可以绕过认证，我们可以在token中放字符串
			//我们想要拉黑一个用户
			var err error
			secret, err = cache.get(kid)
			if err != nil {
				return nil, ErrMissingSecret
			}

			return []byte(secret.Key), nil
		}, jwt.WithAudience(AuthzAudience), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !parsedT.Valid {
			reject(c, cache.failureResponder, ErrInvalidToken)
			return
		}

		if KeyExpired(secret.Expires) {
			reject(c, cache.failureResponder, ErrExpiredCredentials)
			return
		}

		c.Set(middlewares.UsernameKey, secret.Username)
		c.Next()
	}
}

// KeyExpired checks if a key has expired, if the value of user.SessionState.Expires is 0, it will be ignored.
func KeyExpired(expires int64) bool {
	if expires >= 1 {
		return time.Now().After(time.Unix(expires, 0))
	}

	return false
}
