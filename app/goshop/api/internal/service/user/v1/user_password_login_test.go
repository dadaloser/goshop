package v1

import (
	"context"
	"testing"
	"time"

	gpb "goshop/api/goods/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/authz"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/errors"
)

func TestPasswordLoginRejectsLockedIdentifierBeforeLookup(t *testing.T) {
	users := &fakeUserData{}
	attempts := &fakeLoginAttempts{locked: true}
	svc := newPasswordLoginTestService(users, attempts)

	_, err := svc.PasswordLogin(context.Background(), "user@example.com", "secret")

	if !errors.IsCode(err, code.ErrUserLoginLocked) {
		t.Fatalf("PasswordLogin() error = %v, want ErrUserLoginLocked", err)
	}
	if users.getByUsernameCalled {
		t.Fatal("PasswordLogin() queried user store for locked identifier")
	}
}

func TestPasswordLoginRecordsFailureForMissingUser(t *testing.T) {
	users := &fakeUserData{
		getByUsernameErr: errors.WithCode(code.ErrUserNotFound, "not found"),
	}
	attempts := &fakeLoginAttempts{}
	svc := newPasswordLoginTestService(users, attempts)

	_, err := svc.PasswordLogin(context.Background(), " USER@example.COM ", "secret")

	if !errors.IsCode(err, code.ErrUserPasswordIncorrect) {
		t.Fatalf("PasswordLogin() error = %v, want ErrUserPasswordIncorrect", err)
	}
	if attempts.recordIdentifier != "user@example.com" {
		t.Fatalf("recorded identifier = %q, want user@example.com", attempts.recordIdentifier)
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
		checkPasswordErr: errors.WithCode(code.ErrUserPasswordIncorrect, "bad password"),
	}
	attempts := &fakeLoginAttempts{recordLocked: true}
	svc := newPasswordLoginTestService(users, attempts)

	_, err := svc.PasswordLogin(context.Background(), "user@example.com", "bad")

	if !errors.IsCode(err, code.ErrUserLoginLocked) {
		t.Fatalf("PasswordLogin() error = %v, want ErrUserLoginLocked", err)
	}
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
	svc := newPasswordLoginTestService(users, attempts)

	got, err := svc.PasswordLogin(context.Background(), " USER_001 ", "secret")

	if err != nil {
		t.Fatalf("PasswordLogin() error = %v", err)
	}
	if got.Token == "" {
		t.Fatal("PasswordLogin() token is empty")
	}
	claims, err := middlewares.NewJWT("01234567890123456789012345678901").ParseToken(got.Token)
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
			if !errors.IsCode(err, code.ErrUserAccountInactive) {
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
	return NewUserService(
		&fakeDataFactory{users: users},
		&options.JwtOptions{
			Realm:      "test",
			Key:        "01234567890123456789012345678901",
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
		},
		nil,
		attempts,
		nil,
		nil,
	)
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
	recordIdentifier string
	resetIdentifier  string
}

func (f *fakeLoginAttempts) IsLocked(context.Context, string) (bool, error) {
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
