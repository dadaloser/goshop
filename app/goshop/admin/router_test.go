package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"goshop/app/goshop/admin/config"
	"goshop/app/pkg/authz"

	"github.com/gin-gonic/gin"
)

func TestRequireAdminToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		opts       *config.AdminAuthOptions
		header     string
		wantStatus int
	}{
		{
			name:       "nil options rejects",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing configured token rejects",
			opts:       &config.AdminAuthOptions{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong token rejects",
			opts:       &config.AdminAuthOptions{Token: "secret"},
			header:     "wrong",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid token passes",
			opts:       &config.AdminAuthOptions{Token: "secret"},
			header:     "secret",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/admin", requireAdminToken(tt.opts), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if tt.header != "" {
				req.Header.Set("X-Admin-Token", tt.header)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireAdminPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		opts       *config.AdminAuthOptions
		permission authz.Permission
		wantStatus int
	}{
		{
			name:       "nil options rejects",
			permission: "user:list",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "missing permission rejects",
			opts:       &config.AdminAuthOptions{Permissions: []string{"goods:list"}},
			permission: "user:list",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "exact permission passes",
			opts:       &config.AdminAuthOptions{Permissions: []string{"user:list"}},
			permission: "user:list",
			wantStatus: http.StatusOK,
		},
		{
			name:       "global wildcard passes",
			opts:       &config.AdminAuthOptions{Permissions: []string{"*"}},
			permission: "user:list",
			wantStatus: http.StatusOK,
		},
		{
			name:       "resource wildcard passes",
			opts:       &config.AdminAuthOptions{Permissions: []string{"user:*"}},
			permission: "user:list",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/admin", requireAdminPermission(tt.opts, tt.permission), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireAdminAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		opts       *config.AdminAuthOptions
		permission authz.Permission
		minRole    string
		wantStatus int
	}{
		{
			name:       "nil options rejects",
			permission: "user:list",
			minRole:    config.AdminRoleAdmin,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "missing permission rejects",
			opts: &config.AdminAuthOptions{
				Role:        config.AdminRoleSuperAdmin,
				Permissions: []string{"goods:list"},
			},
			permission: "user:list",
			minRole:    config.AdminRoleAdmin,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "insufficient role rejects wildcard permission",
			opts: &config.AdminAuthOptions{
				Role:        config.AdminRoleBasic,
				Permissions: []string{"*"},
			},
			permission: "user:list",
			minRole:    config.AdminRoleAdmin,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "permission and role pass",
			opts: &config.AdminAuthOptions{
				Role:        config.AdminRoleAdmin,
				Permissions: []string{string(authz.PermissionUserListAny)},
			},
			permission: authz.PermissionUserListAny,
			minRole:    config.AdminRoleAdmin,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/admin", requireAdminAccess(tt.opts, tt.permission, tt.minRole), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireAdminConfirmation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		opts       *config.AdminAuthOptions
		header     string
		wantStatus int
	}{
		{
			name:       "missing configured token rejects",
			opts:       &config.AdminAuthOptions{},
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "wrong confirmation rejects",
			opts:       &config.AdminAuthOptions{ConfirmationToken: "confirm-secret"},
			header:     "wrong",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "valid confirmation passes",
			opts:       &config.AdminAuthOptions{ConfirmationToken: "confirm-secret"},
			header:     "confirm-secret",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/admin", requireAdminConfirmation(tt.opts), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/admin", nil)
			if tt.header != "" {
				req.Header.Set("X-Admin-Confirm-Token", tt.header)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestAdminAuthChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		opts       *config.AdminAuthOptions
		header     string
		wantStatus int
	}{
		{
			name:       "valid token without permission rejects",
			opts:       &config.AdminAuthOptions{Token: "secret"},
			header:     "secret",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "wrong token with permission rejects",
			opts:       &config.AdminAuthOptions{Token: "secret", Permissions: []string{string(authz.PermissionUserListAny)}},
			header:     "wrong",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid token and permission passes",
			opts:       &config.AdminAuthOptions{Token: "secret", Permissions: []string{string(authz.PermissionUserListAny)}},
			header:     "secret",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/admin",
				requireAdminToken(tt.opts),
				requireAdminPermission(tt.opts, authz.PermissionUserListAny),
				func(c *gin.Context) {
					c.Status(http.StatusOK)
				},
			)

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if tt.header != "" {
				req.Header.Set("X-Admin-Token", tt.header)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
