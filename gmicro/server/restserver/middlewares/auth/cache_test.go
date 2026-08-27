package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestCacheStrategyDoesNotExposeInternalParseError(t *testing.T) {
	const audience = "api.example.test"
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var failure error
	router.Use(NewCacheStrategy(func(kid string) (Secret, error) {
		return Secret{}, errors.New("backend secret lookup failed")
	}, audience, WithFailureResponder(func(c *gin.Context, err error) {
		failure = err
		c.Status(http.StatusUnauthorized)
	})).AuthFunc())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": audience,
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

	if !errors.Is(failure, ErrInvalidToken) {
		t.Fatalf("authentication failure = %v, want %v", failure, ErrInvalidToken)
	}
	if errors.Is(failure, ErrMissingSecret) {
		t.Fatalf("authentication failure leaked internal cache error: %v", failure)
	}
}

func TestCacheStrategyRejectsMissingAudience(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	secretLookupCalled := false
	router.Use(NewCacheStrategy(func(string) (Secret, error) {
		secretLookupCalled = true
		return Secret{Key: "secret"}, nil
	}, "").AuthFunc())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"aud": "api.example.test"})
	token.Header["kid"] = "key-1"
	rawToken, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if secretLookupCalled {
		t.Fatal("secret lookup was called without a configured audience")
	}
}
