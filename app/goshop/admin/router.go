package admin

import (
	"crypto/subtle"
	"net/http"
	"time"

	goodspb "goshop/api/goods/v1"
	inventorypb "goshop/api/inventory/v1"
	orderpb "goshop/api/order/v1"
	upbv1 "goshop/api/user/v1"
	"goshop/app/goshop/admin/config"
	"goshop/app/goshop/admin/controller"
	"goshop/app/pkg/authsession/tokenrevocation"
	"goshop/app/pkg/authsession/tokenversion"
	"goshop/app/pkg/authz"
	"goshop/gmicro/server/restserver"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 初始化路由
func initRouter(g *restserver.Server, cfg *config.Config, users upbv1.UserClient) error {
	return initRouterWithBusinessClients(g, cfg, users, nil, nil, nil)
}

func initRouterWithBusinessClients(g *restserver.Server, cfg *config.Config, users upbv1.UserClient, goods goodspb.GoodsClient, inventory inventorypb.InventoryClient, orders orderpb.OrderClient) error {
	return initRouterWithDependencies(g, cfg, users, goods, inventory, orders, tokenrevocation.NewRedisStore(), tokenversion.NewRedisStore())
}

func initRouterWithSessionStores(
	g *restserver.Server,
	cfg *config.Config,
	users upbv1.UserClient,
	revokedTokens tokenrevocation.Store,
	tokenVersions tokenversion.Store,
) error {
	return initRouterWithDependencies(g, cfg, users, nil, nil, nil, revokedTokens, tokenVersions)
}

func initRouterWithDependencies(
	g *restserver.Server, cfg *config.Config, users upbv1.UserClient,
	goods goodspb.GoodsClient, inventory inventorypb.InventoryClient, orders orderpb.OrderClient,
	revokedTokens tokenrevocation.Store, tokenVersions tokenversion.Store,
) error {
	if cfg != nil && cfg.Server != nil && cfg.Server.ManagementPort > 0 {
		registerBusinessLivez(g)
	}
	v1 := g.Group("/v1")
	staffAuth, err := newStaffJWTAuth(cfg.Jwt, revokedTokens, tokenVersions, users)
	if err != nil {
		return err
	}
	ucontroller := controller.NewUserController(users, tokenVersions)
	operations := newOperationsHandler(users, goods, inventory, orders)
	authController := newStaffAuthHandler(users, cfg.Jwt, cfg.AdminAuth, revokedTokens, tokenVersions)
	v1.POST("/auth/login", authController.Login)
	v1.POST("/auth/logout", staffAuth.AuthFunc(), authz.RequirePrincipalTypes(authz.PrincipalStaff), authController.Logout)
	v1.POST("/auth/logout_all", staffAuth.AuthFunc(), authz.RequirePrincipalTypes(authz.PrincipalStaff), authController.LogoutAll)
	v1.GET("/auth/me", staffAuth.AuthFunc(), authz.RequirePrincipalTypes(authz.PrincipalStaff), authController.Me)
	v1.POST("/break_glass/session", requireAdminToken(cfg.AdminAuth), authController.BootstrapSession)
	v1.GET("/admin/audit_logs", staffAuth.AuthFunc(), authz.RequirePrincipalTypes(authz.PrincipalStaff), authz.RequirePermission(authz.PermissionAuditReadAny), ucontroller.ListAdminAuditLogs)

	ugroup := v1.Group("/user", staffAuth.AuthFunc(), authz.RequirePrincipalTypes(authz.PrincipalStaff))
	ugroup.POST("staff", authz.RequirePermission(authz.PermissionUserCreateAny), requireAdminConfirmation(cfg.AdminAuth), ucontroller.CreateStaff)
	ugroup.GET("list", authz.RequirePermission(authz.PermissionUserListAny), ucontroller.List)
	ugroup.GET(":id", authz.RequirePermission(authz.PermissionUserReadAny), ucontroller.GetByID)
	ugroup.PUT(":id/status", authz.RequirePermission(authz.PermissionUserDisableAny), requireAdminConfirmation(cfg.AdminAuth), ucontroller.UpdateStatus)
	ugroup.GET(":id/audit_logs", authz.RequirePermission(authz.PermissionAuditReadAny), ucontroller.ListAuditLogs)
	ugroup.GET(":id/roles", authz.RequirePermission(authz.PermissionRoleReadAny), ucontroller.GetUserStaffRoles)
	ugroup.PUT(":id/roles", authz.RequirePermission(authz.PermissionRoleAssignAny), requireAdminConfirmation(cfg.AdminAuth), ucontroller.ReplaceUserStaffRoles)
	ugroup.PUT(":id/resource_scopes", authz.RequirePermission(authz.PermissionRoleAssignAny), requireRole(authz.StaffRoleSuperAdmin), requireResourceScope(authz.BusinessDomainPlatform), requireAdminConfirmation(cfg.AdminAuth), ucontroller.ReplaceResourceScopes)
	staffGroup := v1.Group("/staff", staffAuth.AuthFunc(), authz.RequirePrincipalTypes(authz.PrincipalStaff))
	staffGroup.GET("roles", authz.RequirePermission(authz.PermissionRoleReadAny), ucontroller.ListStaffRoles)
	staffGroup.POST("roles", authz.RequirePermission(authz.PermissionRoleWriteAny), requireAdminConfirmation(cfg.AdminAuth), ucontroller.CreateStaffRole)
	staffGroup.PUT("roles/:name", authz.RequirePermission(authz.PermissionRoleWriteAny), requireAdminConfirmation(cfg.AdminAuth), ucontroller.UpdateStaffRole)
	staffGroup.DELETE("roles/:name", authz.RequirePermission(authz.PermissionRoleWriteAny), requireAdminConfirmation(cfg.AdminAuth), ucontroller.DeleteStaffRole)
	staffGroup.GET("permission_templates", authz.RequirePermission(authz.PermissionRoleReadAny), ucontroller.ListPermissionTemplates)

	registerOperationsRoutes(v1, staffAuth, cfg, operations)
	return nil
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

		provided := c.GetHeader(headerName)
		currentValid := adminTokenEqual(expected, provided)
		previousValid := opts != nil && opts.PreviousTokenActive(time.Now()) && adminTokenEqual(opts.EffectivePreviousToken(), provided)
		if !currentValid && !previousValid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "invalid admin token",
			})
			return
		}
		c.Next()
	}
}

func requireAdminPermission(opts *config.AdminAuthOptions, permission authz.Permission) gin.HandlerFunc {
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

func requireAdminAccess(opts *config.AdminAuthOptions, permission authz.Permission, minRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if opts == nil || !opts.HasAccess(permission, minRole) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":       http.StatusForbidden,
				"msg":        "admin access denied",
				"permission": permission,
				"min_role":   minRole,
			})
			return
		}
		c.Next()
	}
}

func requireAdminConfirmation(opts *config.AdminAuthOptions) gin.HandlerFunc {
	const headerName = "X-Admin-Confirm-Token"

	return func(c *gin.Context) {
		expected := ""
		if opts != nil {
			expected = opts.EffectiveConfirmationToken()
		}
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code":   http.StatusServiceUnavailable,
				"msg":    "admin confirmation is not configured",
				"detail": "set admin-auth.confirmation-token or GOSHOP_ADMIN_CONFIRMATION_TOKEN before exposing high-risk admin write routes",
			})
			return
		}
		if !adminTokenEqual(expected, c.GetHeader(headerName)) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": http.StatusForbidden,
				"msg":  "admin confirmation required",
			})
			return
		}
		c.Set(operationCorrelationKey, uuid.NewString())
		c.Next()
	}
}

func adminTokenEqual(expected, got string) bool {
	if expected == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(got)) == 1
}

func registerBusinessLivez(g *restserver.Server) {
	g.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
