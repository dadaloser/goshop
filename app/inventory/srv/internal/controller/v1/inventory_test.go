package v1

import (
	"context"
	"testing"

	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

func TestInventoryServerRejectsNilRequests(t *testing.T) {
	server := &inventoryServer{}

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "set inventory",
			run: func() error {
				_, err := server.SetInv(context.Background(), nil)
				return err
			},
		},
		{
			name: "inventory detail",
			run: func() error {
				_, err := server.InvDetail(context.Background(), nil)
				return err
			},
		},
		{
			name: "get stock",
			run: func() error {
				_, err := server.GetStock(context.Background(), nil)
				return err
			},
		},
		{
			name: "sell",
			run: func() error {
				_, err := server.Sell(context.Background(), nil)
				return err
			},
		},
		{
			name: "reserve",
			run: func() error {
				_, err := server.Reserve(context.Background(), nil)
				return err
			},
		},
		{
			name: "reback",
			run: func() error {
				_, err := server.Reback(context.Background(), nil)
				return err
			},
		},
		{
			name: "confirm",
			run: func() error {
				_, err := server.Confirm(context.Background(), nil)
				return err
			},
		},
		{
			name: "release",
			run: func() error {
				_, err := server.Release(context.Background(), nil)
				return err
			},
		},
		{
			name: "set stock",
			run: func() error {
				_, err := server.SetStock(context.Background(), nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, code2.ErrValidation) {
				t.Fatalf("error = %v, want code %d", err, code2.ErrValidation)
			}
		})
	}
}
