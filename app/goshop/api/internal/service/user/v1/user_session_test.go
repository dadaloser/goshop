package v1

import (
	"context"
	"goshop/app/pkg/bizcode"
	"goshop/gmicro/errcode"
	"testing"

	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/authz"
	"goshop/pkg/errors"
)

func TestSessionErrorsExposeSpecs(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
		code int
		kind errors.Kind
	}{
		{
			name: "invalid refresh token",
			run: func() error {
				svc := NewUserService(nil, validJWTOptions(), nil, nil, nil, nil)
				_, err := svc.Refresh(context.Background(), "", "")
				return err
			},
			code: errcode.ErrTokenInvalid,
			kind: errors.KindUnauthenticated,
		},
		{
			name: "missing jwt options",
			run: func() error {
				var svc *userService
				_, _, _, _, _, err := svc.createToken(context.Background(), data.UserAuth{})
				return err
			},
			code: bizcode.ErrConnectGRPC,
			kind: errors.KindUnavailable,
		},
		{
			name: "inactive account",
			run: func() error {
				svc := NewUserService(nil, validJWTOptions(), nil, nil, nil, nil).(*userService)
				_, _, _, _, _, err := svc.createToken(context.Background(), data.UserAuth{
					User: data.User{Status: string(authz.AccountStatusDisabled)},
				})
				return err
			},
			code: bizcode.ErrUserAccountInactive,
			kind: errors.KindPermissionDenied,
		},
		{
			name: "missing user for logout all",
			run: func() error {
				svc := NewUserService(nil, validJWTOptions(), nil, nil, nil, nil)
				return svc.LogoutAll(context.Background(), 0)
			},
			code: bizcode.ErrUserNotFound,
			kind: errors.KindNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want code %d", err, tt.code)
			}
			spec, ok := errors.SpecOf(err)
			if !ok {
				t.Fatal("error has no specification")
			}
			if spec.Kind != tt.kind {
				t.Fatalf("error kind = %q, want %q", spec.Kind, tt.kind)
			}
		})
	}
}

func TestEmailAuthRejectsUnavailableVerificationStore(t *testing.T) {
	svc := NewUserService(nil, validJWTOptions(), nil, nil, nil, nil).(*userService)
	svc.emailCodes = nil

	for _, call := range []struct {
		name string
		run  func() error
	}{
		{
			name: "login",
			run: func() error {
				_, err := svc.EmailLogin(context.Background(), "user@example.com", "123456")
				return err
			},
		},
		{
			name: "registration",
			run: func() error {
				_, err := svc.EmailRegister(context.Background(), "13800138000", "user@example.com", "user", "password", "User", "123456")
				return err
			},
		},
	} {
		t.Run(call.name, func(t *testing.T) {
			err := call.run()
			if !errors.IsCode(err, bizcode.ErrEmailVerificationUnavailable) {
				t.Fatalf("error = %v, want code %d", err, bizcode.ErrEmailVerificationUnavailable)
			}
			requireErrorKind(t, err, errors.KindUnavailable)
		})
	}
}

func requireErrorKind(t *testing.T, err error, want errors.Kind) {
	t.Helper()

	spec, ok := errors.SpecOf(err)
	if !ok {
		t.Fatal("error has no specification")
	}
	if spec.Kind != want {
		t.Fatalf("error kind = %q, want %q", spec.Kind, want)
	}
}
