package api

import (
	"fmt"

	"goshop/app/goshop/api/internal/data"
	"goshop/app/goshop/api/internal/tokenrevocation"
	"goshop/app/goshop/api/internal/tokenversion"
	"goshop/app/pkg/authz"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/gmicro/server/restserver/middlewares/auth"
	"goshop/pkg/log"

	"github.com/gin-gonic/gin"
)

// 可以在此处使用别的中间件来实现认证授权
func newJWTAuth(opts *options.JwtOptions, revokedTokens tokenrevocation.Store, tokenVersions tokenversion.Store, users data.UserData) (middlewares.AuthStrategy, error) {
	if opts == nil {
		return nil, fmt.Errorf("jwt options are required")
	}
	parser := middlewares.NewJWT(opts.Key)
	return auth.NewJWTStrategy([]byte(opts.Key), opts.Realm, middlewares.KeyUserID, func(_ interface{}, c *gin.Context) bool {
		rawToken, ok := c.Get(middlewares.JWTTokenKey)
		if !ok {
			return false
		}
		token, ok := rawToken.(string)
		if !ok {
			return false
		}

		if revokedTokens != nil {
			revoked, err := revokedTokens.IsRevoked(c.Request.Context(), token)
			if err != nil {
				log.Errorf("check jwt revocation failed: %v", err)
				return false
			}
			if revoked {
				return false
			}
		}

		claims, err := parser.ParseToken(token)
		if err != nil {
			log.Errorf("parse jwt claims failed: %v", err)
			return false
		}

		if users == nil {
			return false
		}
		user, err := users.Get(c.Request.Context(), uint64(claims.ID))
		if err != nil {
			log.Errorf("check user account status failed: %v", err)
			return false
		}
		if authz.NormalizeAccountStatus(user.Status) != authz.AccountStatusActive {
			return false
		}
		if tokenVersions == nil {
			return true
		}
		currentVersion, err := tokenVersions.CurrentVersion(c.Request.Context(), uint64(claims.ID))
		if err != nil {
			log.Errorf("check jwt token version failed: %v", err)
			return false
		}
		return currentVersion == claims.TokenVersion
	}), nil
}
