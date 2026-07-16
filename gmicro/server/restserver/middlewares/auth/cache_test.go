package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"goshop/gmicro/code"
	pkgerrors "goshop/pkg/errors"
)

func TestCacheStrategyDoesNotExposeInternalParseError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewCacheStrategy(func(kid string) (Secret, error) {
		return Secret{}, errors.New("backend secret lookup failed")
	}).AuthFunc())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": AuthzAudience,
	})
	token.Header["kid"] = "missing"
	rawToken, err := token.SignedString([]byte("wrong-secret"))
	if err != nil {
		t.Fatalf("SignedString() error = %v, want nil", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, pkgerrors.ParseCoder(pkgerrors.WithCode(code.ErrSignatureInvalid, "")).String()) {
		t.Fatalf("response body = %q, want signature invalid code", body)
	}
	if strings.Contains(body, "backend secret lookup failed") || strings.Contains(body, ErrMissingSecret.Error()) {
		t.Fatalf("response body leaks internal auth error: %q", body)
	}
}
