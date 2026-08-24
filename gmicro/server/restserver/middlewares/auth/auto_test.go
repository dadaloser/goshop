package auth

import (
	stdErrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAutoStrategyPassesInjectedResponderToDelegatedStrategy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var failure error
	strategy := NewAutoStrategy(
		NewBasicStrategy(func(_, _ string) bool { return false }),
		NewJWTStrategy(nil, "", "", "", nil),
		WithFailureResponder(func(c *gin.Context, err error) {
			failure = err
			c.Status(http.StatusUnauthorized)
		}),
	)

	router := gin.New()
	router.Use(strategy.AuthFunc())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjppbnZhbGlk")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("AutoStrategy status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if !stdErrors.Is(failure, ErrInvalidCredentials) {
		t.Errorf("AutoStrategy failure = %v, want %v", failure, ErrInvalidCredentials)
	}
}
