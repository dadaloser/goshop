package v1

import (
	"context"
	"goshop/app/pkg/bizcode"
	"net/http/httptest"
	"testing"
	"time"

	gpb "goshop/api/goods/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/authclaims"
	"goshop/app/pkg/authz"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

func TestPasswordLoginRejectsLockedIdentifierBeforeLookup(t *testing.T) {
	users := &fakeUserData{}
	attempts := &fakeLoginAttempts{locked: true}
	svc := newPasswordLoginTestService(users, attempts)

	_, err := svc.PasswordLogin(context.Background(), "user@example.com", "secret")

	if !errors.IsCode(err, bizcode.ErrUserLoginLocked) {
		t.Fatalf("PasswordLogin() error = %v, want ErrUserLoginLocked", err)
	}
	requireErrorKind(t, err, errors.KindRateLimited)
	if users.getByUsernameCalled {
		t.Fatal("PasswordLogin() queried user store for locked identifier")
	}
	if attempts.lockedIdentifier != "user@example.com" {
		t.Fatalf("IsLocked() identifier = %q, want %q", attempts.lockedIdentifier, "user@example.com")
	}
}

func TestPasswordLoginRejectsLockedIPBeforeLookup(t *testing.T) {
	users := &fakeUserData{}
	accountAttempts := &fakeLoginAttempts{}
	ipAttempts := &fakeLoginAttempts{locked: true}
	svc := newPasswordLoginTestServiceWithIPStore(users, accountAttempts, ipAttempts)

	_, err := svc.PasswordLogin(newClientIPContext("203.0.113.10"), "user@example.com", "secret")

	if !errors.IsCode(err, bizcode.ErrUserLoginLocked) {
		t.Fatalf("PasswordLogin() error = %v, want ErrUserLoginLocked", err)
	}
	if users.getByUsernameCalled {
		t.Fatal("PasswordLogin() queried user store for ip-locked request")
	}
	if ipAttempts.lockedIdentifier != "203.0.113.10" {
		t.Fatalf("IsLocked() ip = %q, want %q", ipAttempts.lockedIdentifier, "203.0.113.10")
	}
}

func TestPasswordLoginRecordsFailureForMissingUser(t *testing.T) {
	users := &fakeUserData{
		getByUsernameErr: errors.NewCode(bizcode.ErrUserNotFound, "not found"),
	}
	attempts := &fakeLoginAttempts{}
	ipAttempts := &fakeLoginAttempts{}
	svc := newPasswordLoginTestServiceWithIPStore(users, attempts, ipAttempts)

	_, err := svc.PasswordLogin(newClientIPContext("198.51.100.7"), " USER@example.COM ", "secret")

	if !errors.IsCode(err, bizcode.ErrUserPasswordIncorrect) {
		t.Fatalf("PasswordLogin() error = %v, want ErrUserPasswordIncorrect", err)
	}
	requireErrorKind(t, err, errors.KindUnauthenticated)
	if attempts.recordIdentifier != "user@example.com" {
		t.Fatalf("recorded identifier = %q, want user@example.com", attempts.recordIdentifier)
	}
	if ipAttempts.recordIdentifier != "198.51.100.7" {
		t.Fatalf("recorded ip = %q, want %q", ipAttempts.recordIdentifier, "198.51.100.7")
	}
}

func TestPasswordLoginReturnsLockedWhenFailureReachesThreshold(t *testing.T) {
	users := &fakeUserData{
		authUser: data.UserAuth{
			User: data.User{
				ID:       1,
				NickName: "tester",
			},
			PasswordHash: "hashed",
		},
		checkPasswordErr: errors.NewCode(bizcode.ErrUserPasswordIncorrect, "bad password"),
	}
	attempts := &fakeLoginAttempts{recordLocked: true}
	svc := newPasswordLoginTestService(users, attempts)

	_, err := svc.PasswordLogin(context.Background(), "user@example.com", "bad")

	if !errors.IsCode(err, bizcode.ErrUserLoginLocked) {
		t.Fatalf("PasswordLogin() error = %v, want ErrUserLoginLocked", err)
	}
	requireErrorKind(t, err, errors.KindRateLimited)
	if attempts.recordIdentifier != "user@example.com" {
		t.Fatalf("recorded identifier = %q, want user@example.com", attempts.recordIdentifier)
	}
}

func TestPasswordLoginResetsFailuresOnSuccess(t *testing.T) {
	users := &fakeUserData{
		authUser: data.UserAuth{
			User: data.User{
				ID:       1,
				NickName: "tester",
			},
			PasswordHash: "hashed",
		},
	}
	attempts := &fakeLoginAttempts{}
	ipAttempts := &fakeLoginAttempts{}
	svc := newPasswordLoginTestServiceWithIPStore(users, attempts, ipAttempts)

	got, err := svc.PasswordLogin(newClientIPContext("192.0.2.44"), " USER_001 ", "secret")

	if err != nil {
		t.Fatalf("PasswordLogin() error = %v", err)
	}
	if got.Token == "" {
		t.Fatal("PasswordLogin() token is empty")
	}
	claims := &authclaims.Claims{}
	err = middlewares.NewJWT("01234567890123456789012345678901").ParseTokenWithClaims(got.Token, claims)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.PrincipalType != string(authz.PrincipalCustomer) {
		t.Fatalf("principal_type = %q, want %q", claims.PrincipalType, authz.PrincipalCustomer)
	}
	if claims.AccountStatus != string(authz.AccountStatusActive) {
		t.Fatalf("status = %q, want %q", claims.AccountStatus, authz.AccountStatusActive)
	}
	if !containsScope(claims.Scope, authz.PermissionOrderReadSelf) {
		t.Fatalf("scope = %#v, want %q", claims.Scope, authz.PermissionOrderReadSelf)
	}
	if users.getAuthUsername != "user_001" {
		t.Fatalf("queried username = %q, want user_001", users.getAuthUsername)
	}
	if attempts.resetIdentifier != "user_001" {
		t.Fatalf("reset identifier = %q, want user_001", attempts.resetIdentifier)
	}
	if ipAttempts.resetIdentifier != "192.0.2.44" {
		t.Fatalf("reset ip = %q, want %q", ipAttempts.resetIdentifier, "192.0.2.44")
	}
}

func TestPasswordLoginRejectsInactiveAccount(t *testing.T) {
	tests := []struct {
		name   string
		status authz.AccountStatus
	}{
		{name: "disabled", status: authz.AccountStatusDisabled},
		{name: "locked", status: authz.AccountStatusLocked},
		{name: "deleted", status: authz.AccountStatusDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := &fakeUserData{user: data.User{
				ID:       1,
				NickName: "tester",
				Status:   string(tt.status),
			}, authUser: data.UserAuth{
				User: data.User{
					ID:       1,
					NickName: "tester",
					Status:   string(tt.status),
				},
				PasswordHash: "hashed",
			}}
			svc := newPasswordLoginTestService(users, &fakeLoginAttempts{})

			got, err := svc.PasswordLogin(context.Background(), "user_001", "secret")
			if !errors.IsCode(err, bizcode.ErrUserAccountInactive) {
				t.Fatalf("PasswordLogin() error = %v, want ErrUserAccountInactive", err)
			}
			if got != nil {
				t.Fatal("PasswordLogin() returned a token for inactive account")
			}
		})
	}
}

func containsScope(scopes []string, permission authz.Permission) bool {
	for _, scope := range scopes {
		if scope == string(permission) {
			return true
		}
	}
	return false
}

func newPasswordLoginTestService(users *fakeUserData, attempts *fakeLoginAttempts) UserSrv {
	return newPasswordLoginTestServiceWithIPStore(users, attempts, &fakeLoginAttempts{})
}

func newPasswordLoginTestServiceWithIPStore(users *fakeUserData, attempts, ipAttempts *fakeLoginAttempts) UserSrv {
	return NewUserServiceWithIPAttempts(
		&fakeDataFactory{users: users},
		&options.JwtOptions{
			Realm:      "test",
			Key:        "01234567890123456789012345678901",
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
		},
		nil,
		attempts,
		ipAttempts,
		nil,
		nil,
	)
}

func newClientIPContext(ip string) *gin.Context {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = ip + ":12345"
	ctx.Request = req
	return ctx
}

type fakeDataFactory struct {
	users data.UserData
}

func (f *fakeDataFactory) Goods() gpb.GoodsClient {
	return nil
}

func (f *fakeDataFactory) Users() data.UserData {
	return f.users
}

type fakeUserData struct {
	user                data.User
	authUser            data.UserAuth
	getByUsernameErr    error
	checkPasswordErr    error
	updateCalled        bool
	updatedUser         *data.User
	created             data.UserCreate
	deleteCalled        bool
	deletedUserID       uint64
	getCalled           bool
	getAuthCalled       bool
	gotAuthID           uint64
	gotID               uint64
	gotUsername         string
	getByUsernameCalled bool
	getAuthUsername     string
}

func (f *fakeUserData) Create(_ context.Context, user *data.UserCreate) (data.User, error) {
	if user != nil {
		f.created = *user
		return data.User{
			ID:         1,
			Username:   user.Username,
			Mobile:     user.Mobile,
			Email:      user.Email,
			NickName:   user.NickName,
			LegacyRole: int32(authz.LegacyUserRoleCustomer),
			Status:     string(authz.AccountStatusActive),
		}, nil
	}
	return data.User{}, nil
}

func (f *fakeUserData) Update(context.Context, *data.User) error {
	f.updateCalled = true
	return nil
}

func (f *fakeUserData) Delete(_ context.Context, userID uint64) error {
	f.deleteCalled = true
	f.deletedUserID = userID
	return nil
}

func (f *fakeUserData) Get(_ context.Context, userID uint64) (data.User, error) {
	f.getCalled = true
	f.gotID = userID
	return f.user, nil
}

func (f *fakeUserData) GetByMobile(context.Context, string) (data.User, error) {
	return data.User{}, nil
}

func (f *fakeUserData) GetByUsername(_ context.Context, username string) (data.User, error) {
	f.getByUsernameCalled = true
	f.gotUsername = username
	if f.getByUsernameErr != nil {
		return data.User{}, f.getByUsernameErr
	}
	return f.user, nil
}

func (f *fakeUserData) GetAuth(_ context.Context, userID uint64) (data.UserAuth, error) {
	f.getAuthCalled = true
	f.gotAuthID = userID
	return f.authUser, nil
}

func (f *fakeUserData) GetAuthByUsername(_ context.Context, username string) (data.UserAuth, error) {
	f.getByUsernameCalled = true
	f.getAuthUsername = username
	if f.getByUsernameErr != nil {
		return data.UserAuth{}, f.getByUsernameErr
	}
	return f.authUser, nil
}

func (f *fakeUserData) CheckPassWord(context.Context, string, string) error {
	return f.checkPasswordErr
}

type fakeLoginAttempts struct {
	locked           bool
	recordLocked     bool
	lockedIdentifier string
	recordIdentifier string
	resetIdentifier  string
}

func (f *fakeLoginAttempts) IsLocked(_ context.Context, identifier string) (bool, error) {
	f.lockedIdentifier = identifier
	return f.locked, nil
}

func (f *fakeLoginAttempts) RecordFailure(_ context.Context, identifier string) (bool, error) {
	f.recordIdentifier = identifier
	return f.recordLocked, nil
}

func (f *fakeLoginAttempts) Reset(_ context.Context, identifier string) error {
	f.resetIdentifier = identifier
	return nil
}

var _ data.DataFactory = &fakeDataFactory{}
var _ data.UserData = &fakeUserData{}
