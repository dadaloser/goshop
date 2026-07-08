package db

import (
	"context"
	"testing"

	"goshop/app/pkg/code"
	dv1 "goshop/app/user/srv/internal/data/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

func TestUsersRejectInvalidLookupBeforeDatabase(t *testing.T) {
	store := &users{}

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "empty mobile",
			run: func() error {
				_, err := store.GetByMobile(context.Background(), " ")
				return err
			},
		},
		{
			name: "empty username",
			run: func() error {
				_, err := store.GetByUsername(context.Background(), " ")
				return err
			},
		},
		{
			name: "zero id",
			run: func() error {
				_, err := store.GetByID(context.Background(), 0)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, code.ErrUserNotFound) {
				t.Fatalf("error = %v, want code %d", err, code.ErrUserNotFound)
			}
		})
	}
}

func TestUsersRejectInvalidWriteBeforeDatabase(t *testing.T) {
	store := &users{}

	tests := []struct {
		name string
		run  func() error
		code int
	}{
		{
			name: "create nil user",
			run: func() error {
				return store.Create(context.Background(), nil)
			},
			code: code2.ErrValidation,
		},
		{
			name: "update nil user",
			run: func() error {
				return store.Update(context.Background(), nil)
			},
			code: code.ErrUserNotFound,
		},
		{
			name: "update zero id",
			run: func() error {
				return store.Update(context.Background(), &dv1.UserDO{})
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
}
