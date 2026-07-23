package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"goshop/app/goshop/admin/config"
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
