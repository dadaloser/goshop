package v1

import (
	"context"
	"testing"
	"time"

	ipb "goshop/api/inventory/v1"
	opb "goshop/api/order/v1"
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

func TestUserServiceRejectsMissingDataDependency(t *testing.T) {
	tests := []struct {
		name string
		svc  UserSrv
		run  func(UserSrv) error
	}{
		{
			name: "nil service get",
			svc:  (*userService)(nil),
			run: func(svc UserSrv) error {
				_, err := svc.Get(context.Background(), 1)
				return err
			},
		},
		{
			name: "nil data factory get",
			svc:  NewUserService(nil, validJWTOptions(), nil, nil, nil, nil),
			run: func(svc UserSrv) error {
				_, err := svc.Get(context.Background(), 1)
				return err
			},
		},
		{
			name: "nil user data update",
			svc:  NewUserService(&fakeDataFactory{}, validJWTOptions(), nil, nil, nil, nil),
			run: func(svc UserSrv) error {
				return svc.Update(context.Background(), &UserDTO{User: data.User{ID: 1}})
			},
		},
		{
			name: "nil user data password login",
			svc:  NewUserService(&fakeDataFactory{}, validJWTOptions(), nil, nil, nil, nil),
			run: func(svc UserSrv) error {
				_, err := svc.PasswordLogin(context.Background(), "user_001", "secret")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(tt.svc)
			if !errors.IsCode(err, code.ErrConnectGRPC) {
				t.Fatalf("error = %v, want code %d", err, code.ErrConnectGRPC)
			}
		})
	}
}

func TestUserServiceRejectsMissingCodeStore(t *testing.T) {
	svc := NewUserService(&fakeDataFactory{users: &fakeUserData{}}, validJWTOptions(), nil, nil, nil, nil)

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "sms login",
			run: func() error {
				_, err := svc.SmsLogin(context.Background(), "13800138000", "123456")
				return err
			},
		},
		{
			name: "register",
			run: func() error {
				_, err := svc.Register(context.Background(), "13800138000", "user@example.com", "user_001", "Strong1!", "tester", "123456")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, code.ErrConnectGRPC) {
				t.Fatalf("error = %v, want code %d", err, code.ErrConnectGRPC)
			}
		})
	}
}

func TestUserServiceRejectsMissingJWTOptions(t *testing.T) {
	users := &fakeUserData{
		user: data.User{
			ID:       1,
			NickName: "tester",
			PassWord: "hashed",
		},
	}
	svc := NewUserService(&fakeDataFactory{users: users}, nil, nil, nil, nil, nil)

	_, err := svc.PasswordLogin(context.Background(), "user_001", "secret")
	if !errors.IsCode(err, code.ErrConnectGRPC) {
		t.Fatalf("PasswordLogin() error = %v, want code %d", err, code.ErrConnectGRPC)
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
		validJWTOptions(),
		nil,
		nil,
		nil,
		nil,
	)
}

func validJWTOptions() *options.JwtOptions {
	return &options.JwtOptions{
		Realm:      "test",
		Key:        "01234567890123456789012345678901",
		Timeout:    time.Hour,
		MaxRefresh: time.Hour,
	}
}

func (f *fakeDataFactory) Orders() opb.OrderClient {
	return nil
}

func (f *fakeDataFactory) Inventory() ipb.InventoryClient {
	return nil
}
