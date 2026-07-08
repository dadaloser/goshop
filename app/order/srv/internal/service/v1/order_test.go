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

func TestListIgnoresMissingUser(t *testing.T) {
	svc := &orderService{}

	got, err := svc.List(context.Background(), 0, metav1.ListMeta{}, nil)
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if got.TotalCount != 0 || len(got.Items) != 0 {
		t.Fatalf("List() = %+v, want empty result", got)
	}
}

func TestCartItemListIgnoresMissingUser(t *testing.T) {
	svc := &orderService{}

	got, err := svc.CartItemList(context.Background(), 0, metav1.ListMeta{}, nil)
	if err != nil {
		t.Fatalf("CartItemList() error = %v, want nil", err)
	}
	if got.TotalCount != 0 || len(got.Items) != 0 {
		t.Fatalf("CartItemList() = %+v, want empty result", got)
	}
}

func TestDeleteCartItemRequiresUser(t *testing.T) {
	svc := &orderService{}

	tests := []struct {
		name   string
		userID uint64
		id     uint64
	}{
		{name: "missing user", id: 1},
		{name: "missing cart id", userID: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.DeleteCartItem(context.Background(), tt.userID, tt.id)
			if !errors.IsCode(err, code.ErrShopCartItemNotFound) {
				t.Fatalf("DeleteCartItem() error = %v, want code %d", err, code.ErrShopCartItemNotFound)
			}
		})
	}
}

func TestDeleteCartItemScopesByUser(t *testing.T) {
	var gotUserID, gotID uint64
	svc := &orderService{
		data: fakeOrderDataFactory{
			shopCarts: fakeShopCartStore{
				delete: func(_ context.Context, userID, id uint64) error {
					gotUserID = userID
					gotID = id
					return nil
				},
			},
		},
	}

	if err := svc.DeleteCartItem(context.Background(), 10, 20); err != nil {
		t.Fatalf("DeleteCartItem() error = %v, want nil", err)
	}
	if gotUserID != 10 || gotID != 20 {
		t.Fatalf("DeleteCartItem() passed userID=%d id=%d, want userID=10 id=20", gotUserID, gotID)
	}
}

func TestUpdateValidatesOrderStatus(t *testing.T) {
	tests := []struct {
		name           string
		currentStatus  string
		nextStatus     string
		wantErr        bool
		wantUpdateCall bool
	}{
		{
			name:           "empty current accepts known status",
			nextStatus:     OrderStatusWaitBuyerPay,
			wantUpdateCall: true,
		},
		{
			name:       "unknown status rejected",
			nextStatus: "UNKNOWN",
			wantErr:    true,
		},
		{
			name:          "closed cannot reopen",
			currentStatus: OrderStatusTradeClosed,
			nextStatus:    OrderStatusWaitBuyerPay,
			wantErr:       true,
		},
		{
			name:          "finished cannot change",
			currentStatus: OrderStatusTradeFinished,
			nextStatus:    OrderStatusTradeClosed,
			wantErr:       true,
		},
		{
			name:           "success can finish",
			currentStatus:  OrderStatusTradeSuccess,
			nextStatus:     OrderStatusTradeFinished,
			wantUpdateCall: true,
		},
		{
			name:          "success cannot go back to paying",
			currentStatus: OrderStatusTradeSuccess,
			nextStatus:    OrderStatusPaying,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var updatedStatus string
			svc := &orderService{
				data: fakeOrderDataFactory{
					orders: fakeOrderStore{
						get: func(context.Context, string) (*do.OrderInfoDO, error) {
							return &do.OrderInfoDO{
								OrderSn: "order-1",
								Status:  tt.currentStatus,
							}, nil
						},
						update: func(_ context.Context, _ *gorm.DB, order *do.OrderInfoDO) error {
							updatedStatus = order.Status
							return nil
						},
					},
				},
			}

			err := svc.Update(context.Background(), &dto.OrderDTO{
				OrderInfoDO: do.OrderInfoDO{
					OrderSn: " order-1 ",
					Status:  tt.nextStatus,
				},
			})
			if tt.wantErr {
				if !errors.IsCode(err, code.ErrOrderStatusInvalid) {
					t.Fatalf("Update() error = %v, want code %d", err, code.ErrOrderStatusInvalid)
				}
				if updatedStatus != "" {
					t.Fatalf("Update() called data update with status %q, want no update", updatedStatus)
				}
				return
			}
			if err != nil {
				t.Fatalf("Update() error = %v, want nil", err)
			}
			if tt.wantUpdateCall && updatedStatus != tt.nextStatus {
				t.Fatalf("Update() updated status = %q, want %q", updatedStatus, tt.nextStatus)
			}
		})
	}
}

func TestUpdateRequiresOrderStatusFields(t *testing.T) {
	svc := &orderService{}

	tests := []struct {
		name  string
		order *dto.OrderDTO
	}{
		{
			name: "nil order",
		},
		{
			name:  "empty order sn",
			order: &dto.OrderDTO{OrderInfoDO: do.OrderInfoDO{Status: OrderStatusPaying}},
		},
		{
			name:  "empty status",
			order: &dto.OrderDTO{OrderInfoDO: do.OrderInfoDO{OrderSn: "order-1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Update(context.Background(), tt.order)
			if !errors.IsCode(err, code.ErrOrderStatusInvalid) {
				t.Fatalf("Update() error = %v, want code %d", err, code.ErrOrderStatusInvalid)
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
	get    func(context.Context, string) (*do.OrderInfoDO, error)
	update func(context.Context, *gorm.DB, *do.OrderInfoDO) error
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

func (f fakeOrderStore) Update(ctx context.Context, txn *gorm.DB, order *do.OrderInfoDO) error {
	if f.update != nil {
		return f.update(ctx, txn, order)
	}
	return nil
}

type fakeShopCartStore struct {
	delete func(context.Context, uint64, uint64) error
}

func (fakeShopCartStore) List(context.Context, uint64, bool, metav1.ListMeta, []string) (*do.ShoppingCartDOList, error) {
	return nil, nil
}

func (fakeShopCartStore) Create(context.Context, *do.ShoppingCartDO) error {
	return nil
}

func (fakeShopCartStore) Get(context.Context, uint64, uint64) (*do.ShoppingCartDO, error) {
	return nil, errors.WithCode(code.ErrShopCartItemNotFound, "shop cart item not found")
}

func (fakeShopCartStore) UpdateNum(context.Context, *do.ShoppingCartDO) error {
	return nil
}

func (f fakeShopCartStore) Delete(ctx context.Context, userID, id uint64) error {
	if f.delete != nil {
		return f.delete(ctx, userID, id)
	}
	return nil
}

func (fakeShopCartStore) ClearCheck(context.Context, uint64) error {
	return nil
}

func (fakeShopCartStore) DeleteByGoodsIDs(context.Context, *gorm.DB, uint64, []int32) error {
	return nil
}

func (fakeShopCartStore) RestoreCheckedItems(context.Context, *gorm.DB, uint64, []*do.OrderGoods) error {
	return nil
}
