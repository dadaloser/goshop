package auth

import (
	stdErrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBasicStrategyRejectsMalformedBase64Authorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var failure error
	router.Use(NewBasicStrategy(func(username string, password string) bool {
		t.Fatalf("compare should not be called for malformed base64 credentials")
		return true
	}, WithFailureResponder(func(c *gin.Context, err error) {
		failure = err
		c.Status(http.StatusUnauthorized)
	})).AuthFunc())
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
	if !stdErrors.Is(failure, ErrInvalidCredentials) {
		t.Fatalf("authentication failure = %v, want %v", failure, ErrInvalidCredentials)
	}
}
