package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goshop/app/goshop/api/internal/tokenrevocation"
	"goshop/app/goshop/api/internal/tokenversion"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/gmicro/server/restserver/middlewares/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuthorizerRejectsTokenVersionMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	strategy, err := newJWTAuth(
		&options.JwtOptions{
			Realm:      "test",
			Key:        "01234567890123456789012345678901",
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
		},
		&fakeRevocationStore{},
		&fakeAuthTokenVersionStore{currentVersion: 2},
	)
	if err != nil {
		t.Fatalf("newJWTAuth() error = %v", err)
	}

	jwtStrategy, ok := strategy.(auth.JWTStrategy)
	if !ok {
		t.Fatalf("strategy type = %T, want auth.JWTStrategy", strategy)
	}

	token, err := middlewares.NewJWT("01234567890123456789012345678901").CreateToken(middlewares.CustomClaims{
		ID:           1,
		TokenVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "test",
		},
	})
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/user/detail", nil)
	ctx.Set(middlewares.JWTTokenKey, token)

	if jwtStrategy.Authorizator(nil, ctx) {
		t.Fatal("Authorizator() = true, want false")
	}
}

type fakeRevocationStore struct{}

func (f *fakeRevocationStore) Revoke(context.Context, string, time.Time) error {
	return nil
}

func (f *fakeRevocationStore) IsRevoked(context.Context, string) (bool, error) {
	return false, nil
}

var _ tokenrevocation.Store = &fakeRevocationStore{}
var _ tokenversion.Store = &fakeAuthTokenVersionStore{}

type fakeAuthTokenVersionStore struct {
	currentVersion uint64
}

func (f *fakeAuthTokenVersionStore) CurrentVersion(context.Context, uint64) (uint64, error) {
	return f.currentVersion, nil
}

func (f *fakeAuthTokenVersionStore) Bump(context.Context, uint64) (uint64, error) {
	return f.currentVersion + 1, nil
}
