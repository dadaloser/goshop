package v1

import (
	"context"
	"testing"

	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/inventory/srv/internal/domain/dto"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

func TestValidateStockOperation(t *testing.T) {
	tests := []struct {
		name    string
		orderSn string
		details []do.GoodsDetail
		wantErr bool
	}{
		{
			name:    "valid detail passes",
			orderSn: "order-001",
			details: []do.GoodsDetail{
				{Goods: 1, Num: 2},
			},
		},
		{
			name:    "empty order rejects",
			orderSn: " ",
			details: []do.GoodsDetail{
				{Goods: 1, Num: 2},
			},
			wantErr: true,
		},
		{
			name:    "empty details rejects",
			orderSn: "order-001",
			wantErr: true,
		},
		{
			name:    "zero goods rejects",
			orderSn: "order-001",
			details: []do.GoodsDetail{
				{Goods: 0, Num: 2},
			},
			wantErr: true,
		},
		{
			name:    "zero quantity rejects",
			orderSn: "order-001",
			details: []do.GoodsDetail{
				{Goods: 1, Num: 0},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStockOperation(tt.orderSn, tt.details)
			if tt.wantErr {
				if !errors.IsCode(err, code2.ErrValidation) {
					t.Fatalf("validateStockOperation() error = %v, want ErrValidation", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateStockOperation() error = %v", err)
			}
		})
	}
}

func TestInventoryServiceRejectsInvalidCreateAndGet(t *testing.T) {
	srv := &inventoryService{}

	tests := []struct {
		name string
		run  func() error
		code int
	}{
		{
			name: "create nil inventory",
			run: func() error {
				return srv.Create(context.Background(), nil)
			},
			code: code2.ErrValidation,
		},
		{
			name: "create zero goods",
			run: func() error {
				return srv.Create(context.Background(), &dto.InventoryDTO{})
			},
			code: code2.ErrValidation,
		},
		{
			name: "create negative stock",
			run: func() error {
				return srv.Create(context.Background(), &dto.InventoryDTO{InventoryDO: do.InventoryDO{Goods: 1, Stocks: -1}})
			},
			code: code2.ErrValidation,
		},
		{
			name: "get zero goods",
			run: func() error {
				_, err := srv.Get(context.Background(), 0)
				return err
			},
			code: code.ErrInventoryNotFound,
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
