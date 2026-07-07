package admin

import (
	"crypto/subtle"
	"net/http"

	"goshop/app/goshop/admin/config"
	"goshop/app/goshop/admin/controller"
	"goshop/gmicro/server/restserver"

	"github.com/gin-gonic/gin"
)

// 初始化路由
func initRouter(g *restserver.Server, cfg *config.Config) {
	v1 := g.Group("/v1")
	adminAuth := requireAdminToken(cfg.AdminAuth)
	ugroup := v1.Group("/user", adminAuth)
	ucontroller := controller.NewUserController()
	ugroup.GET("list", requireAdminPermission(cfg.AdminAuth, "user:list"), ucontroller.List)
}

func requireAdminToken(opts *config.AdminAuthOptions) gin.HandlerFunc {
	const headerName = "X-Admin-Token"

	return func(c *gin.Context) {
		expected := ""
		if opts != nil {
			expected = opts.EffectiveToken()
		}
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":   http.StatusUnauthorized,
				"msg":    "admin auth is not configured",
				"detail": "set admin-auth.token or GOSHOP_ADMIN_TOKEN before exposing admin routes",
			})
			return
		}

		if !adminTokenEqual(expected, c.GetHeader(headerName)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "invalid admin token",
			})
			return
		}
		c.Next()
	}
}

func requireAdminPermission(opts *config.AdminAuthOptions, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if opts == nil || !opts.HasPermission(permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":       http.StatusForbidden,
				"msg":        "admin permission denied",
				"permission": permission,
			})
			return
		}
		c.Next()
	}
}

func adminTokenEqual(expected, got string) bool {
	if expected == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(got)) == 1
}
