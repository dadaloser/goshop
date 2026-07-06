package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"goshop/app/goshop/admin/config"

	"github.com/gin-gonic/gin"
)

func TestRequireAdminToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		configured string
		header     string
		wantStatus int
	}{
		{
			name:       "missing configured token rejects",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong token rejects",
			configured: "secret",
			header:     "wrong",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid token passes",
			configured: "secret",
			header:     "secret",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/admin", requireAdminToken(&config.AdminAuthOptions{Token: tt.configured}), func(c *gin.Context) {
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
