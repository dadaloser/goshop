package v1

import (
	"context"
	"testing"
	"time"

	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

func TestUserServiceRejectsInvalidUpdateAndGet(t *testing.T) {
	users := &fakeUserData{}
	svc := newUserBoundaryTestService(users)

	tests := []struct {
		name string
		run  func() error
		code int
	}{
		{
			name: "update nil user",
			run: func() error {
				return svc.Update(context.Background(), nil)
			},
			code: code2.ErrValidation,
		},
		{
			name: "update zero id",
			run: func() error {
				return svc.Update(context.Background(), &UserDTO{})
			},
			code: code2.ErrValidation,
		},
		{
			name: "get zero id",
			run: func() error {
				_, err := svc.Get(context.Background(), 0)
				return err
			},
			code: code.ErrUserNotFound,
		},
		{
			name: "get empty username",
			run: func() error {
				_, err := svc.GetByUsername(context.Background(), " ")
				return err
			},
			code: code.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want code %d", err, tt.code)
			}
		})
	}
	if users.updateCalled {
		t.Fatal("invalid update reached user data store")
	}
	if users.getCalled || users.getByUsernameCalled {
		t.Fatal("invalid get reached user data store")
	}
}

func TestUserServiceNormalizesGetByUsername(t *testing.T) {
	users := &fakeUserData{
		user: data.User{ID: 1, NickName: "tester"},
	}
	svc := newUserBoundaryTestService(users)

	if _, err := svc.GetByUsername(context.Background(), " USER@example.COM "); err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if users.gotUsername != "user@example.com" {
		t.Fatalf("queried username = %q, want user@example.com", users.gotUsername)
	}
}

func newUserBoundaryTestService(users *fakeUserData) UserSrv {
	return NewUserService(
		&fakeDataFactory{users: users},
		&options.JwtOptions{
			Realm:      "test",
			Key:        "01234567890123456789012345678901",
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
		},
		nil,
		nil,
		nil,
	)
}
