package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"goshop/gmicro/server/restserver/middlewares"
)

func TestLogoutRevokesCurrentToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeRevocationStore{}
	server := &userServer{revokedTokens: store}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/user/logout", nil)
	expiresAt := time.Now().Add(time.Hour).Truncate(time.Second)
	ctx.Request.Header.Set("Authorization", "Bearer raw-token")
	ctx.Set(middlewares.JWTPayloadKey, map[string]any{
		"exp": float64(expiresAt.Unix()),
	})

	server.Logout(ctx)

	if store.token != "raw-token" {
		t.Fatalf("revoked token = %q, want raw-token", store.token)
	}
	if !store.expiresAt.Equal(expiresAt) {
		t.Fatalf("revoked expiresAt = %v, want %v", store.expiresAt, expiresAt)
	}
}

func TestJWTExpiresAtRejectsMissingExp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set(middlewares.JWTPayloadKey, map[string]any{})

	if _, err := jwtExpiresAt(ctx); err == nil {
		t.Fatal("jwtExpiresAt() error = nil, want error")
	}
}

type fakeRevocationStore struct {
	token     string
	expiresAt time.Time
}

func (f *fakeRevocationStore) Revoke(_ context.Context, token string, expiresAt time.Time) error {
	f.token = token
	f.expiresAt = expiresAt
	return nil
}

func (f *fakeRevocationStore) IsRevoked(context.Context, string) (bool, error) {
	return false, nil
}
