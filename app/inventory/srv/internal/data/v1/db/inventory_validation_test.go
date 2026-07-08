package mysql

import (
	"context"
	"testing"

	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

func TestInventoryStoreRejectsInvalidInputBeforeDatabase(t *testing.T) {
	store := &inventorys{}

	tests := []struct {
		name string
		run  func() error
		code int
	}{
		{
			name: "get zero goods",
			run: func() error {
				_, err := store.Get(context.Background(), 0)
				return err
			},
			code: code.ErrInventoryNotFound,
		},
		{
			name: "reduce zero goods",
			run: func() error {
				return store.Reduce(context.Background(), nil, 0, 1)
			},
			code: code.ErrInventoryNotFound,
		},
		{
			name: "reduce negative quantity",
			run: func() error {
				return store.Reduce(context.Background(), nil, 1, -1)
			},
			code: code2.ErrValidation,
		},
		{
			name: "increase zero quantity",
			run: func() error {
				return store.Increase(context.Background(), nil, 1, 0)
			},
			code: code2.ErrValidation,
		},
		{
			name: "create nil inventory",
			run: func() error {
				return store.Create(context.Background(), nil)
			},
			code: code2.ErrValidation,
		},
		{
			name: "create negative stock",
			run: func() error {
				return store.Create(context.Background(), &do.InventoryDO{Goods: 1, Stocks: -1})
			},
			code: code2.ErrValidation,
		},
		{
			name: "get sell detail empty order",
			run: func() error {
				_, err := store.GetSellDetail(context.Background(), nil, " ")
				return err
			},
			code: code.ErrInvSellDetailNotFound,
		},
		{
			name: "update sell detail empty order",
			run: func() error {
				return store.UpdateStockSellDetailStatus(context.Background(), nil, " ", 1)
			},
			code: code.ErrInvSellDetailNotFound,
		},
		{
			name: "update sell detail invalid status",
			run: func() error {
				return store.UpdateStockSellDetailStatus(context.Background(), nil, "order-1", 0)
			},
			code: code2.ErrValidation,
		},
		{
			name: "create nil sell detail",
			run: func() error {
				return store.CreateStockSellDetail(context.Background(), nil, nil)
			},
			code: code.ErrInvSellDetailNotFound,
		},
		{
			name: "create sell detail invalid item",
			run: func() error {
				return store.CreateStockSellDetail(context.Background(), nil, &do.StockSellDetailDO{
					OrderSn: "order-1",
					Status:  1,
					Detail:  do.GoodsDetailList{{Goods: 1}},
				})
			},
			code: code2.ErrValidation,
		},
		{
			name: "create sell detail if absent invalid status",
			run: func() error {
				_, err := store.CreateStockSellDetailIfAbsent(context.Background(), nil, &do.StockSellDetailDO{
					OrderSn: "order-1",
					Detail:  do.GoodsDetailList{{Goods: 1, Num: 1}},
				})
				return err
			},
			code: code2.ErrValidation,
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
