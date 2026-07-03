package api

import (
	"fmt"

	"goshop/app/pkg/options"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/gmicro/server/restserver/middlewares/auth"

	"github.com/gin-gonic/gin"

	ginjwt "github.com/appleboy/gin-jwt/v2"
)

// 可以在此处使用别的中间件来实现认证授权
func newJWTAuth(opts *options.JwtOptions) (middlewares.AuthStrategy, error) {
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
		TokenLookup:     "header: Authorization:, query: token, cookie: jwt",
		TokenHeadName:   "Bearer",
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
