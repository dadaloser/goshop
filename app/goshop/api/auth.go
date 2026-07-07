package api

import (
	"fmt"

	"goshop/app/goshop/api/internal/tokenrevocation"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/gmicro/server/restserver/middlewares/auth"
	"goshop/pkg/log"

	"github.com/gin-gonic/gin"

	ginjwt "github.com/appleboy/gin-jwt/v2"
)

// 可以在此处使用别的中间件来实现认证授权
func newJWTAuth(opts *options.JwtOptions, revokedTokens tokenrevocation.Store) (middlewares.AuthStrategy, error) {
	if opts == nil {
		return nil, fmt.Errorf("jwt options are required")
	}
	gjwt, err := ginjwt.New(&ginjwt.GinJWTMiddleware{
		Realm:            opts.Realm,
		SigningAlgorithm: "HS256",
		Key:              []byte(opts.Key),
		Timeout:          opts.Timeout,
		MaxRefresh:       opts.MaxRefresh,
		LogoutResponse: func(c *gin.Context, code int) {
			c.JSON(code, nil)
		},
		IdentityHandler: claimHandlerFun,
		IdentityKey:     middlewares.KeyUserID,
		Authorizator: func(_ interface{}, c *gin.Context) bool {
			if revokedTokens == nil {
				return true
			}
			rawToken, ok := c.Get("JWT_TOKEN")
			if !ok {
				return false
			}
			token, ok := rawToken.(string)
			if !ok {
				return false
			}
			revoked, err := revokedTokens.IsRevoked(c.Request.Context(), token)
			if err != nil {
				log.Errorf("check jwt revocation failed: %v", err)
				return false
			}
			return !revoked
		},
		TokenLookup:   "header: Authorization:, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
	})
	if err != nil {
		return nil, fmt.Errorf("create jwt middleware: %w", err)
	}
	return auth.NewJWTStrategy(*gjwt), nil
}

func claimHandlerFun(c *gin.Context) interface{} {
	claims := ginjwt.ExtractClaims(c)
	c.Set(middlewares.KeyUserID, claims[middlewares.KeyUserID])
	return claims[ginjwt.IdentityKey]
}
