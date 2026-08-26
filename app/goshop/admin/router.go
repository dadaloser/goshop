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
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"
	"goshop/pkg/storage"
	core "goshop/pkg/transport/httperror"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 初始化路由
func initRouterWithBusinessClients(g *restserver.Server, cfg *config.Config, users upbv1.UserClient, goods goodspb.GoodsClient, inventory inventorypb.InventoryClient, orders orderpb.OrderClient, clients ...*storage.Client) error {
	return initRouterWithDependencies(g, cfg, users, goods, inventory, orders, tokenrevocation.NewRedisStore(clients...), tokenversion.NewRedisStore(clients...))
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
	v1.POST("/break_glass/approvals", requireAdminToken(cfg.AdminAuth), authController.CreateBreakGlassApproval)
	v1.POST("/break_glass/approvals/:approval_id/approve", requireAdminToken(cfg.AdminAuth), requireAdminConfirmation(cfg.AdminAuth), authController.ApproveBreakGlassApproval)
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
	staffGroup.GET("sessions", requireRole(authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireResourceScope(authz.BusinessDomainPlatform), authz.RequirePermission(authz.PermissionStaffSessionReadAny), ucontroller.ListStaffSessions)
	staffGroup.POST("sessions/:session_id/revoke", requireRole(authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireTargetResourceScope(authz.BusinessDomainPlatform, "staff_session", "session_id"), authz.RequirePermission(authz.PermissionStaffSessionRevokeAny), requireAdminConfirmation(cfg.AdminAuth), ucontroller.RevokeStaffSession)
	staffGroup.POST(":id/sessions/revoke", requireRole(authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireTargetResourceScope(authz.BusinessDomainPlatform, "staff_user", "id"), authz.RequirePermission(authz.PermissionStaffSessionRevokeAny), requireAdminConfirmation(cfg.AdminAuth), ucontroller.RevokeStaffUserSessions)
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
			core.AbortWithError(c, apperrors.NewCode(errcode.ErrServiceUnavailable, "admin authentication is not configured"))
			return
		}

		provided := c.GetHeader(headerName)
		currentValid := adminTokenEqual(expected, provided)
		previousValid := opts != nil && opts.PreviousTokenActive(time.Now()) && adminTokenEqual(opts.EffectivePreviousToken(), provided)
		if !currentValid && !previousValid {
			core.AbortWithError(c, apperrors.NewCode(errcode.ErrTokenInvalid, "invalid admin token"))
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
			core.AbortWithError(c, apperrors.NewCode(errcode.ErrServiceUnavailable, "admin confirmation is not configured"))
			return
		}
		if !adminTokenEqual(expected, c.GetHeader(headerName)) {
			core.AbortWithError(c, apperrors.NewCode(errcode.ErrPermissionDenied, "admin confirmation required"))
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
