package v1

import (
	"context"
	"testing"
	"time"

	gpb "goshop/api/goods/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
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
		user: data.User{
			ID:       1,
			NickName: "tester",
			PassWord: "hashed",
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
		user: data.User{
			ID:       1,
			NickName: "tester",
			PassWord: "hashed",
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
	if users.gotUsername != "user_001" {
		t.Fatalf("queried username = %q, want user_001", users.gotUsername)
	}
	if attempts.resetIdentifier != "user_001" {
		t.Fatalf("reset identifier = %q, want user_001", attempts.resetIdentifier)
	}
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
	getByUsernameErr    error
	checkPasswordErr    error
	gotUsername         string
	getByUsernameCalled bool
}

func (f *fakeUserData) Create(context.Context, *data.User) error {
	return nil
}

func (f *fakeUserData) Update(context.Context, *data.User) error {
	return nil
}

func (f *fakeUserData) Get(context.Context, uint64) (data.User, error) {
	return data.User{}, nil
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
