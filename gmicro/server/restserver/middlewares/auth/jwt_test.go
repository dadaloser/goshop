package auth

import (
	"net/http/httptest"
	"testing"
	"time"

	"goshop/gmicro/server/restserver/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTStrategyRequiresConfiguredAudience(t *testing.T) {
	const key = "01234567890123456789012345678901"
	rawToken, err := middlewares.NewJWT(key).CreateToken(middlewares.CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{"goshop-api"},
			Issuer:    "test",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	if err != nil {
		t.Fatalf("CreateToken() error = %v, want nil", err)
	}

	apiStrategy := NewJWTStrategy([]byte(key), "test", "goshop-api", middlewares.KeyUserID, nil)
	if _, err := apiStrategy.parseToken(rawToken); err != nil {
		t.Errorf("parseToken() API audience error = %v, want nil", err)
	}

	adminStrategy := NewJWTStrategy([]byte(key), "test", "goshop-admin", middlewares.KeyUserID, nil)
	if _, err := adminStrategy.parseToken(rawToken); err == nil {
		t.Error("parseToken() cross-audience error = nil, want audience rejection")
	}
}

func TestGetTokenRejectsQueryParameter(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/?token=secret", nil)

	if token, err := GetToken(ctx); err == nil {
		t.Fatalf("GetToken() = %q, nil, want query token rejection", token)
	}
}

func TestGetTokenAcceptsBearerHeader(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer secret")

	token, err := GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken() error = %v, want nil", err)
	}
	if token != "secret" {
		t.Errorf("GetToken() = %q, want %q", token, "secret")
	}
}
