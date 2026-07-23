package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/authsession/tokenrevocation"
	"goshop/app/pkg/authsession/tokenversion"
	"goshop/app/pkg/authz"
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
		&fakeAuthUserStore{user: data.User{ID: 1, Status: string(authz.AccountStatusActive)}},
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

func TestJWTAuthorizerChecksCurrentAccountStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		status authz.AccountStatus
		getErr error
		want   bool
	}{
		{name: "active account passes", status: authz.AccountStatusActive, want: true},
		{name: "disabled account rejects", status: authz.AccountStatusDisabled},
		{name: "locked account rejects", status: authz.AccountStatusLocked},
		{name: "deleted account rejects", status: authz.AccountStatusDeleted},
		{name: "user lookup failure rejects", getErr: errors.New("rpc unavailable")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const key = "01234567890123456789012345678901"
			strategy, err := newJWTAuth(
				&options.JwtOptions{Realm: "test", Key: key, Timeout: time.Hour, MaxRefresh: time.Hour},
				&fakeRevocationStore{},
				&fakeAuthTokenVersionStore{currentVersion: 1},
				&fakeAuthUserStore{user: data.User{ID: 1, Status: string(tt.status)}, getErr: tt.getErr},
			)
			if err != nil {
				t.Fatalf("newJWTAuth() error = %v", err)
			}
			jwtStrategy := strategy.(auth.JWTStrategy)
			token, err := middlewares.NewJWT(key).CreateToken(middlewares.CustomClaims{
				ID: 1, TokenVersion: 1,
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

			if got := jwtStrategy.Authorizator(nil, ctx); got != tt.want {
				t.Fatalf("Authorizator() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestJWTAuthorizerRejectsRevokedDeviceSession(t *testing.T) {
	const key = "01234567890123456789012345678901"
	users := &fakeSessionAuthUserStore{fakeAuthUserStore: fakeAuthUserStore{user: data.User{ID: 1, Status: string(authz.AccountStatusActive)}}}
	strategy, err := newJWTAuth(&options.JwtOptions{Realm: "test", Key: key, Timeout: time.Hour, MaxRefresh: time.Hour}, &fakeRevocationStore{}, &fakeAuthTokenVersionStore{currentVersion: 1}, users)
	if err != nil {
		t.Fatalf("newJWTAuth() error = %v", err)
	}
	token, err := middlewares.NewJWT(key).CreateToken(middlewares.CustomClaims{ID: 1, TokenVersion: 1, SessionID: "device-session", RegisteredClaims: jwt.RegisteredClaims{NotBefore: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), Issuer: "test"}})
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/user/detail", nil)
	ctx.Set(middlewares.JWTTokenKey, token)
	if strategy.(auth.JWTStrategy).Authorizator(nil, ctx) {
		t.Fatal("revoked session authorized")
	}
	users.active = true
	if !strategy.(auth.JWTStrategy).Authorizator(nil, ctx) {
		t.Fatal("active session rejected")
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

type fakeAuthUserStore struct {
	user   data.User
	getErr error
}

type fakeSessionAuthUserStore struct {
	fakeAuthUserStore
	active bool
}

func (f *fakeSessionAuthUserStore) ValidateSession(context.Context, uint64, string) (bool, error) {
	return f.active, nil
}

func (f *fakeAuthUserStore) Create(context.Context, *data.UserCreate) (data.User, error) {
	return data.User{}, nil
}
func (f *fakeAuthUserStore) Update(context.Context, *data.User) error { return nil }
func (f *fakeAuthUserStore) Delete(context.Context, uint64) error     { return nil }
func (f *fakeAuthUserStore) Get(context.Context, uint64) (data.User, error) {
	return f.user, f.getErr
}
func (f *fakeAuthUserStore) GetByMobile(context.Context, string) (data.User, error) {
	return f.user, f.getErr
}
func (f *fakeAuthUserStore) GetByUsername(context.Context, string) (data.User, error) {
	return f.user, f.getErr
}
func (f *fakeAuthUserStore) GetAuth(context.Context, uint64) (data.UserAuth, error) {
	return data.UserAuth{User: f.user}, f.getErr
}
func (f *fakeAuthUserStore) GetAuthByUsername(context.Context, string) (data.UserAuth, error) {
	return data.UserAuth{User: f.user}, f.getErr
}
func (f *fakeAuthUserStore) CheckPassWord(context.Context, string, string) error { return nil }

func (f *fakeAuthTokenVersionStore) CurrentVersion(context.Context, uint64) (uint64, error) {
	return f.currentVersion, nil
}

func (f *fakeAuthTokenVersionStore) Bump(context.Context, uint64) (uint64, error) {
	return f.currentVersion + 1, nil
}
