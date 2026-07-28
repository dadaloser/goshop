package db

import (
	"context"
	"testing"

	"goshop/app/order/srv/internal/domain/do"
	"goshop/app/pkg/bizcode"
	"goshop/pkg/errors"
)

func TestOrdersRejectInvalidInputBeforeDatabase(t *testing.T) {
	store := &orders{}

	tests := []struct {
		name string
		run  func() error
		code int
	}{
		{
			name: "get empty order sn",
			run: func() error {
				_, err := store.Get(context.Background(), " ")
				return err
			},
			code: bizcode.ErrOrderNotFound,
		},
		{
			name: "create nil order",
			run: func() error {
				return store.Create(context.Background(), nil, nil)
			},
			code: bizcode.ErrSubmitOrder,
		},
		{
			name: "create invalid identity",
			run: func() error {
				return store.Create(context.Background(), nil, &do.OrderInfoDO{OrderSn: "order-1"})
			},
			code: bizcode.ErrSubmitOrder,
		},
		{
			name: "update nil order",
			run: func() error {
				return store.Update(context.Background(), nil, nil)
			},
			code: bizcode.ErrOrderNotFound,
		},
		{
			name: "update missing identity",
			run: func() error {
				return store.Update(context.Background(), nil, &do.OrderInfoDO{Status: "PAYING"})
			},
			code: bizcode.ErrOrderNotFound,
		},
		{
			name: "update missing status",
			run: func() error {
				return store.Update(context.Background(), nil, &do.OrderInfoDO{OrderSn: "order-1"})
			},
			code: bizcode.ErrOrderStatusInvalid,
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

func TestShopCartsRejectInvalidInputBeforeDatabase(t *testing.T) {
	store := &shopCarts{}

	tests := []struct {
		name string
		run  func() error
		code int
	}{
		{
			name: "create nil cart",
			run: func() error {
				return store.Create(context.Background(), nil)
			},
			code: bizcode.ErrShopCartItemNotFound,
		},
		{
			name: "create invalid cart",
			run: func() error {
				return store.Create(context.Background(), &do.ShoppingCartDO{User: 1, Goods: 1})
			},
			code: bizcode.ErrShopCartItemNotFound,
		},
		{
			name: "get missing user",
			run: func() error {
				_, err := store.Get(context.Background(), 0, 1)
				return err
			},
			code: bizcode.ErrShopCartItemNotFound,
		},
		{
			name: "get missing goods",
			run: func() error {
				_, err := store.Get(context.Background(), 1, 0)
				return err
			},
			code: bizcode.ErrShopCartItemNotFound,
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
