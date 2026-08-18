package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersSetsDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q, want no-referrer", got)
	}
	if got := rec.Header().Get("Permissions-Policy"); got == "" {
		t.Fatal("Permissions-Policy should be set")
	}
	if got := rec.Header().Get("Content-Security-Policy"); got != "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'" {
		t.Fatalf("Content-Security-Policy = %q, want restrictive default policy", got)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got != "max-age=31536000" {
		t.Fatalf("Strict-Transport-Security = %q, want max-age=31536000", got)
	}
}

func TestSecurityHeadersAllowsCustomCSP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeadersWithOptions(SecurityHeadersOptions{
		ContentSecurityPolicy: "default-src 'self'",
	}))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Security-Policy"); got != "default-src 'self'" {
		t.Fatalf("Content-Security-Policy = %q, want custom policy", got)
	}
}

func TestSecurityHeadersApplyToAbortedPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders(), CorsWithOptions(CorsOptions{AllowOrigins: []string{"https://shop.example.com"}}))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://shop.example.com")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("Content-Security-Policy is empty on preflight response")
	}
}
