package service

import (
	"context"
	"testing"

	datav1 "goshop/app/order/srv/internal/data/v1"
	"goshop/app/order/srv/internal/domain/do"
	"goshop/app/order/srv/internal/domain/dto"
	"goshop/app/pkg/code"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"

	"gorm.io/gorm"
)

func TestCreateComIgnoresEmptyOrder(t *testing.T) {
	svc := &orderService{}

	tests := []struct {
		name  string
		order *dto.OrderDTO
	}{
		{
			name: "nil order",
		},
		{
			name: "empty order sn",
			order: &dto.OrderDTO{
				OrderInfoDO: do.OrderInfoDO{OrderSn: " "},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := svc.CreateCom(context.Background(), tt.order); err != nil {
				t.Fatalf("CreateCom() error = %v", err)
			}
		})
	}
}

func TestCreateRejectsEmptyOrderGoods(t *testing.T) {
	svc := &orderService{}

	err := svc.Create(context.Background(), &dto.OrderDTO{
		OrderInfoDO: do.OrderInfoDO{OrderSn: "order-1"},
	})
	if !errors.IsCode(err, code.ErrNoGoodsSelect) {
		t.Fatalf("Create() error = %v, want code %d", err, code.ErrNoGoodsSelect)
	}
}

func TestCreateIsIdempotentForSameOrder(t *testing.T) {
	existing := &do.OrderInfoDO{
		User:         10,
		OrderSn:      "order-1",
		Address:      "address",
		SignerName:   "buyer",
		SingerMobile: "13800138000",
		Post:         "post",
		OrderGoods: []*do.OrderGoods{
			{Goods: 1, Nums: 2},
			{Goods: 2, Nums: 1},
		},
	}
	svc := &orderService{
		data: fakeOrderDataFactory{
			orders: fakeOrderStore{
				get: func(context.Context, string) (*do.OrderInfoDO, error) {
					return existing, nil
				},
			},
		},
	}

	err := svc.Create(context.Background(), &dto.OrderDTO{
		OrderInfoDO: do.OrderInfoDO{
			User:         10,
			OrderSn:      " order-1 ",
			Address:      "address",
			SignerName:   "buyer",
			SingerMobile: "13800138000",
			Post:         "post",
			OrderGoods: []*do.OrderGoods{
				{Goods: 2, Nums: 1},
				{Goods: 1, Nums: 2},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
}

func TestCreateRejectsOrderSnConflict(t *testing.T) {
	existing := &do.OrderInfoDO{
		User:         10,
		OrderSn:      "order-1",
		Address:      "address",
		SignerName:   "buyer",
		SingerMobile: "13800138000",
		Post:         "post",
		OrderGoods: []*do.OrderGoods{
			{Goods: 1, Nums: 2},
		},
	}
	svc := &orderService{
		data: fakeOrderDataFactory{
			orders: fakeOrderStore{
				get: func(context.Context, string) (*do.OrderInfoDO, error) {
					return existing, nil
				},
			},
		},
	}

	err := svc.Create(context.Background(), &dto.OrderDTO{
		OrderInfoDO: do.OrderInfoDO{
			User:         11,
			OrderSn:      "order-1",
			Address:      "address",
			SignerName:   "buyer",
			SingerMobile: "13800138000",
			Post:         "post",
			OrderGoods: []*do.OrderGoods{
				{Goods: 1, Nums: 2},
			},
		},
	})
	if !errors.IsCode(err, code.ErrOrderConflict) {
		t.Fatalf("Create() error = %v, want code %d", err, code.ErrOrderConflict)
	}
}

func TestGetRequiresUserOwnership(t *testing.T) {
	existing := &do.OrderInfoDO{
		User:    10,
		OrderSn: "order-1",
		OrderGoods: []*do.OrderGoods{
			{Goods: 1, Nums: 2},
		},
	}
	svc := &orderService{
		data: fakeOrderDataFactory{
			orders: fakeOrderStore{
				get: func(context.Context, string) (*do.OrderInfoDO, error) {
					return existing, nil
				},
			},
		},
	}

	tests := []struct {
		name    string
		userID  uint64
		orderSn string
		wantErr bool
	}{
		{
			name:    "owner can read",
			userID:  10,
			orderSn: "order-1",
		},
		{
			name:    "missing user is rejected",
			orderSn: "order-1",
			wantErr: true,
		},
		{
			name:    "other user is rejected",
			userID:  11,
			orderSn: "order-1",
			wantErr: true,
		},
		{
			name:    "empty order sn is rejected",
			userID:  10,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Get(context.Background(), tt.userID, tt.orderSn)
			if tt.wantErr {
				if !errors.IsCode(err, code.ErrOrderNotFound) {
					t.Fatalf("Get() error = %v, want code %d", err, code.ErrOrderNotFound)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get() error = %v, want nil", err)
			}
			if got.OrderSn != existing.OrderSn {
				t.Fatalf("Get() order_sn = %q, want %q", got.OrderSn, existing.OrderSn)
			}
		})
	}
}

type fakeOrderDataFactory struct {
	orders    datav1.OrderStore
	shopCarts datav1.ShopCartStore
}

func (f fakeOrderDataFactory) Orders() datav1.OrderStore {
	return f.orders
}

func (f fakeOrderDataFactory) ShopCarts() datav1.ShopCartStore {
	return f.shopCarts
}

func (fakeOrderDataFactory) Begin() *gorm.DB {
	return &gorm.DB{}
}

type fakeOrderStore struct {
	get func(context.Context, string) (*do.OrderInfoDO, error)
}

func (f fakeOrderStore) Get(ctx context.Context, orderSn string) (*do.OrderInfoDO, error) {
	if f.get != nil {
		return f.get(ctx, orderSn)
	}
	return nil, errors.WithCode(code.ErrOrderNotFound, "order not found")
}

func (fakeOrderStore) List(context.Context, uint64, metav1.ListMeta, []string) (*do.OrderInfoDOList, error) {
	return nil, nil
}

func (fakeOrderStore) Create(context.Context, *gorm.DB, *do.OrderInfoDO) error {
	return nil
}

func (fakeOrderStore) DeleteByOrderSn(context.Context, *gorm.DB, string) error {
	return nil
}

func (fakeOrderStore) Update(context.Context, *gorm.DB, *do.OrderInfoDO) error {
	return nil
}
