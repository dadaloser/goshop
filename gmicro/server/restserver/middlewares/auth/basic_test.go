package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"goshop/gmicro/code"
	pkgerrors "goshop/pkg/errors"
)

func TestBasicStrategyRejectsMalformedBase64Authorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewBasicStrategy(func(username string, password string) bool {
		t.Fatalf("compare should not be called for malformed base64 credentials")
		return true
	}).AuthFunc())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic !!!not-base64!!!")
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want authentication failure", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, pkgerrors.ParseCoder(pkgerrors.WithCode(code.ErrSignatureInvalid, "")).String()) {
		t.Fatalf("response body = %q, want signature invalid code", body)
	}
}
