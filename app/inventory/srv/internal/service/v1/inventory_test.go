package v1

import (
	"testing"

	"goshop/app/inventory/srv/internal/domain/do"
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
