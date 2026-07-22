package service

import (
	"context"
	"reflect"
	"testing"
	"time"

	"goshop/app/order/srv/internal/boundary"
	datav1 "goshop/app/order/srv/internal/data/v1"
	"goshop/app/order/srv/internal/domain/do"
	"goshop/app/order/srv/internal/domain/dto"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
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

func TestNormalizeInitialOrderStatus(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "empty defaults to wait buyer pay",
			input:  " ",
			output: OrderStatusWaitBuyerPay,
		},
		{
			name:   "existing status is preserved",
			input:  OrderStatusPaying,
			output: OrderStatusPaying,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeInitialOrderStatus(tt.input); got != tt.output {
				t.Fatalf("normalizeInitialOrderStatus(%q) = %q, want %q", tt.input, got, tt.output)
			}
		})
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

func TestCreateWritesInitialStatusLog(t *testing.T) {
	var logs []*do.OrderStatusLogDO
	svc := &orderService{
		data: fakeOrderDataFactory{
			orders: fakeOrderStore{
				get: func(context.Context, string) (*do.OrderInfoDO, error) {
					return nil, errors.WithCode(code.ErrOrderNotFound, "order not found")
				},
				create: func(_ context.Context, _ *gorm.DB, order *do.OrderInfoDO) error {
					order.ID = 42
					return nil
				},
			},
			orderStatusLogs: fakeOrderStatusLogStore{
				create: func(_ context.Context, _ *gorm.DB, entry *do.OrderStatusLogDO) error {
					logs = append(logs, entry)
					return nil
				},
			},
			shopCarts: fakeShopCartStore{},
		},
		upstream: upstream{
			goods: fakeGoodsGateway{
				batchGetGoods: func(_ context.Context, ids []int32) (map[int32]boundary.GoodsInfo, error) {
					return map[int32]boundary.GoodsInfo{
						101: {ID: 101, Name: "goods-101", ShopPriceFen: 1000, GoodsFrontImage: "img-101"},
					}, nil
				},
			},
		},
	}

	err := svc.Create(context.Background(), &dto.OrderDTO{
		OrderInfoDO: do.OrderInfoDO{
			User:         9,
			OrderSn:      "order-1",
			Address:      "addr",
			SignerName:   "buyer",
			SingerMobile: "13800138000",
			OrderGoods: []*do.OrderGoods{
				{Goods: 101, Nums: 2},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("status logs = %d, want 1", len(logs))
	}
	if logs[0].OrderID != 42 || logs[0].OrderSn != "order-1" || logs[0].ToStatus != OrderStatusWaitBuyerPay || logs[0].Reason != "order created" {
		t.Fatalf("status log = %+v", logs[0])
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

func TestStatusLogsRequiresUserOwnership(t *testing.T) {
	svc := &orderService{
		data: fakeOrderDataFactory{
			orders: fakeOrderStore{
				get: func(context.Context, string) (*do.OrderInfoDO, error) {
					return &do.OrderInfoDO{
						User:    10,
						OrderSn: "order-1",
					}, nil
				},
			},
			orderStatusLogs: fakeOrderStatusLogStore{
				listByOrderSn: func(context.Context, string) ([]*do.OrderStatusLogDO, error) {
					return []*do.OrderStatusLogDO{
						{
							OrderSn:    "order-1",
							FromStatus: OrderStatusWaitBuyerPay,
							ToStatus:   OrderStatusTradeSuccess,
						},
					}, nil
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
		{name: "owner can read", userID: 10, orderSn: "order-1"},
		{name: "missing user", orderSn: "order-1", wantErr: true},
		{name: "other user", userID: 11, orderSn: "order-1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.StatusLogs(context.Background(), tt.userID, tt.orderSn)
			if tt.wantErr {
				if !errors.IsCode(err, code.ErrOrderNotFound) {
					t.Fatalf("StatusLogs() error = %v, want code %d", err, code.ErrOrderNotFound)
				}
				return
			}
			if err != nil {
				t.Fatalf("StatusLogs() error = %v", err)
			}
			if got.TotalCount != 1 || len(got.Items) != 1 || got.Items[0].OrderSn != "order-1" {
				t.Fatalf("StatusLogs() = %+v", got)
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

func TestSubmitRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name  string
		svc   *orderService
		order *dto.OrderDTO
	}{
		{
			name: "nil order",
			svc:  &orderService{},
		},
		{
			name: "missing user",
			svc:  &orderService{dtmOpts: &options.DtmOptions{GrpcServer: "127.0.0.1:36790"}},
			order: &dto.OrderDTO{
				OrderInfoDO: do.OrderInfoDO{OrderSn: "order-1"},
			},
		},
		{
			name: "missing order sn",
			svc:  &orderService{dtmOpts: &options.DtmOptions{GrpcServer: "127.0.0.1:36790"}},
			order: &dto.OrderDTO{
				OrderInfoDO: do.OrderInfoDO{User: 10},
			},
		},
		{
			name: "missing dtm config",
			svc:  &orderService{},
			order: &dto.OrderDTO{
				OrderInfoDO: do.OrderInfoDO{User: 10, OrderSn: "order-1"},
			},
		},
		{
			name: "empty dtm grpc server",
			svc:  &orderService{dtmOpts: &options.DtmOptions{}},
			order: &dto.OrderDTO{
				OrderInfoDO: do.OrderInfoDO{User: 10, OrderSn: "order-1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.svc.Submit(context.Background(), tt.order)
			if !errors.IsCode(err, code.ErrSubmitOrder) {
				t.Fatalf("Submit() error = %v, want code %d", err, code.ErrSubmitOrder)
			}
		})
	}
}

func TestUpdateValidatesOrderStatus(t *testing.T) {
	tests := []struct {
		name           string
		currentStatus  string
		nextStatus     string
		tradeNo        string
		wantErr        bool
		wantUpdateCall bool
	}{
		{
			name:           "empty current accepts known status",
			nextStatus:     OrderStatusWaitBuyerPay,
			wantUpdateCall: true,
		},
		{
			name:           "wait buyer pay can become success",
			currentStatus:  OrderStatusWaitBuyerPay,
			nextStatus:     OrderStatusTradeSuccess,
			tradeNo:        "trade-1",
			wantUpdateCall: true,
		},
		{
			name:           "wait buyer pay can become closed",
			currentStatus:  OrderStatusWaitBuyerPay,
			nextStatus:     OrderStatusTradeClosed,
			wantUpdateCall: true,
		},
		{
			name:          "wait buyer pay cannot finish directly",
			currentStatus: OrderStatusWaitBuyerPay,
			nextStatus:    OrderStatusTradeFinished,
			wantErr:       true,
		},
		{
			name:           "paying can become success",
			currentStatus:  OrderStatusPaying,
			nextStatus:     OrderStatusTradeSuccess,
			tradeNo:        "trade-1",
			wantUpdateCall: true,
		},
		{
			name:          "paying cannot return to wait buyer pay",
			currentStatus: OrderStatusPaying,
			nextStatus:    OrderStatusWaitBuyerPay,
			wantErr:       true,
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
					TradeNo: tt.tradeNo,
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

func TestUpdateCreatesStatusLogOnTransition(t *testing.T) {
	var logs []*do.OrderStatusLogDO
	svc := &orderService{
		data: fakeOrderDataFactory{
			orders: fakeOrderStore{
				get: func(context.Context, string) (*do.OrderInfoDO, error) {
					return &do.OrderInfoDO{
						User:    9,
						OrderSn: "order-1",
						Status:  OrderStatusWaitBuyerPay,
					}, nil
				},
				update: func(_ context.Context, _ *gorm.DB, order *do.OrderInfoDO) error {
					return nil
				},
			},
			orderStatusLogs: fakeOrderStatusLogStore{
				create: func(_ context.Context, _ *gorm.DB, entry *do.OrderStatusLogDO) error {
					logs = append(logs, entry)
					return nil
				},
			},
		},
	}

	err := svc.Update(context.Background(), &dto.OrderDTO{
		OrderInfoDO: do.OrderInfoDO{
			OrderSn: "order-1",
			Status:  OrderStatusTradeClosed,
		},
		StatusReason:   "order timeout auto close",
		StatusSource:   "order.timeout_worker",
		StatusOperator: "system",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("status logs = %d, want 1", len(logs))
	}
	if logs[0].FromStatus != OrderStatusWaitBuyerPay || logs[0].ToStatus != OrderStatusTradeClosed || logs[0].Reason != "order timeout auto close" || logs[0].Source != "order.timeout_worker" || logs[0].Operator != "system" {
		t.Fatalf("status log = %+v", logs[0])
	}
}

func TestUpdateSkipsStatusLogWhenStatusUnchanged(t *testing.T) {
	var logCreated bool
	svc := &orderService{
		data: fakeOrderDataFactory{
			orders: fakeOrderStore{
				get: func(context.Context, string) (*do.OrderInfoDO, error) {
					return &do.OrderInfoDO{
						User:    9,
						OrderSn: "order-1",
						Status:  OrderStatusTradeSuccess,
					}, nil
				},
				update: func(_ context.Context, _ *gorm.DB, order *do.OrderInfoDO) error {
					return nil
				},
			},
			orderStatusLogs: fakeOrderStatusLogStore{
				create: func(_ context.Context, _ *gorm.DB, entry *do.OrderStatusLogDO) error {
					logCreated = true
					return nil
				},
			},
		},
	}

	now := time.Unix(1700000000, 0)
	err := svc.Update(context.Background(), &dto.OrderDTO{
		OrderInfoDO: do.OrderInfoDO{
			OrderSn: "order-1",
			Status:  OrderStatusTradeSuccess,
			TradeNo: "trade-1",
			PayTime: &now,
		},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if logCreated {
		t.Fatal("Update() created status log for unchanged status")
	}
}

func TestProcessExpiredOrdersReleasesInventoryAndClosesOrder(t *testing.T) {
	var released []boundary.InventoryItem
	var updated []*dto.OrderDTO
	now := time.Unix(1700000000, 0)
	svcFactory := &service{
		data: fakeOrderDataFactory{
			orders: fakeOrderStore{
				listClose: func(_ context.Context, statuses []string, createdBefore time.Time, limit int) ([]*do.OrderInfoDO, error) {
					if limit != orderLifecycleBatchSize {
						t.Fatalf("limit = %d, want %d", limit, orderLifecycleBatchSize)
					}
					if !reflect.DeepEqual(statuses, []string{OrderStatusWaitBuyerPay, OrderStatusPaying}) {
						t.Fatalf("statuses = %v", statuses)
					}
					if !createdBefore.Equal(now.Add(-orderTimeoutCloseAfter)) {
						t.Fatalf("createdBefore = %v, want %v", createdBefore, now.Add(-orderTimeoutCloseAfter))
					}
					return []*do.OrderInfoDO{
						{
							User:    9,
							OrderSn: "order-1",
							Status:  OrderStatusWaitBuyerPay,
							OrderGoods: []*do.OrderGoods{
								{Goods: 101, Nums: 2},
							},
						},
					}, nil
				},
				get: func(context.Context, string) (*do.OrderInfoDO, error) {
					return &do.OrderInfoDO{
						User:    9,
						OrderSn: "order-1",
						Status:  OrderStatusWaitBuyerPay,
					}, nil
				},
				update: func(_ context.Context, _ *gorm.DB, order *do.OrderInfoDO) error {
					updated = append(updated, &dto.OrderDTO{OrderInfoDO: *order})
					return nil
				},
			},
			orderStatusLogs: fakeOrderStatusLogStore{},
		},
		upstream: upstream{
			inventory: fakeInventoryGateway{
				release: func(_ context.Context, orderSn string, items []boundary.InventoryItem) error {
					if orderSn != "order-1" {
						t.Fatalf("orderSn = %s, want order-1", orderSn)
					}
					released = append(released, items...)
					return nil
				},
			},
		},
		lifecycle: LifecycleConfig{
			PollInterval:      orderLifecyclePollInterval,
			TimeoutCloseAfter: orderTimeoutCloseAfter,
			BatchSize:         orderLifecycleBatchSize,
		},
		now: func() time.Time { return now },
	}

	if err := svcFactory.processExpiredOrdersOnce(context.Background()); err != nil {
		t.Fatalf("processExpiredOrdersOnce() error = %v", err)
	}
	if !reflect.DeepEqual(released, []boundary.InventoryItem{{GoodsID: 101, Num: 2}}) {
		t.Fatalf("released = %+v", released)
	}
	if len(updated) != 1 || updated[0].Status != OrderStatusTradeClosed {
		t.Fatalf("updated = %+v", updated)
	}
}

func TestProcessFinishedOrdersMarksFinished(t *testing.T) {
	var updated []*dto.OrderDTO
	now := time.Unix(1700000000, 0)
	svcFactory := &service{
		data: fakeOrderDataFactory{
			orders: fakeOrderStore{
				listFinish: func(_ context.Context, status string, paidBefore time.Time, limit int) ([]*do.OrderInfoDO, error) {
					if status != OrderStatusTradeSuccess {
						t.Fatalf("status = %s, want %s", status, OrderStatusTradeSuccess)
					}
					if !paidBefore.Equal(now) {
						t.Fatalf("paidBefore = %v, want %v", paidBefore, now)
					}
					return []*do.OrderInfoDO{
						{
							User:    9,
							OrderSn: "order-2",
							Status:  OrderStatusTradeSuccess,
						},
					}, nil
				},
				get: func(context.Context, string) (*do.OrderInfoDO, error) {
					return &do.OrderInfoDO{
						User:    9,
						OrderSn: "order-2",
						Status:  OrderStatusTradeSuccess,
					}, nil
				},
				update: func(_ context.Context, _ *gorm.DB, order *do.OrderInfoDO) error {
					updated = append(updated, &dto.OrderDTO{OrderInfoDO: *order})
					return nil
				},
			},
			orderStatusLogs: fakeOrderStatusLogStore{},
		},
		lifecycle: LifecycleConfig{
			PollInterval:       orderLifecyclePollInterval,
			TimeoutCloseAfter:  orderTimeoutCloseAfter,
			FinishAfterPayment: orderFinishAfterPayment,
			BatchSize:          orderLifecycleBatchSize,
		},
		now: func() time.Time { return now },
	}

	if err := svcFactory.processFinishedOrdersOnce(context.Background()); err != nil {
		t.Fatalf("processFinishedOrdersOnce() error = %v", err)
	}
	if len(updated) != 1 || updated[0].Status != OrderStatusTradeFinished {
		t.Fatalf("updated = %+v", updated)
	}
}

func TestTransitionMetricName(t *testing.T) {
	tests := []struct {
		name       string
		fromStatus string
		toStatus   string
		want       string
	}{
		{name: "create", toStatus: OrderStatusWaitBuyerPay, want: "create"},
		{name: "noop", fromStatus: OrderStatusTradeSuccess, toStatus: OrderStatusTradeSuccess, want: "noop"},
		{name: "pay success", fromStatus: OrderStatusWaitBuyerPay, toStatus: OrderStatusTradeSuccess, want: "pay_success"},
		{name: "close", fromStatus: OrderStatusWaitBuyerPay, toStatus: OrderStatusTradeClosed, want: "close"},
		{name: "finish", fromStatus: OrderStatusTradeSuccess, toStatus: OrderStatusTradeFinished, want: "finish"},
		{name: "change", fromStatus: OrderStatusWaitBuyerPay, toStatus: OrderStatusPaying, want: "change"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transitionMetricName(tt.fromStatus, tt.toStatus); got != tt.want {
				t.Fatalf("transitionMetricName(%q, %q) = %q, want %q", tt.fromStatus, tt.toStatus, got, tt.want)
			}
		})
	}
}

func TestNormalizeTransitionTrigger(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: "update"},
		{input: "order.create", want: "create"},
		{input: "order.payment", want: "payment"},
		{input: "order.timeout_worker", want: "timeout_worker"},
		{input: "order.finish_worker", want: "finish_worker"},
		{input: "something-else", want: "custom"},
	}

	for _, tt := range tests {
		if got := normalizeTransitionTrigger(tt.input); got != tt.want {
			t.Fatalf("normalizeTransitionTrigger(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

type fakeOrderDataFactory struct {
	orders          datav1.OrderStore
	orderStatusLogs datav1.OrderStatusLogStore
	shopCarts       datav1.ShopCartStore
}

func (f fakeOrderDataFactory) Orders() datav1.OrderStore {
	return f.orders
}

func (f fakeOrderDataFactory) OrderStatusLogs() datav1.OrderStatusLogStore {
	return f.orderStatusLogs
}

func (f fakeOrderDataFactory) ShopCarts() datav1.ShopCartStore {
	return f.shopCarts
}

func (fakeOrderDataFactory) Begin() *gorm.DB {
	return nil
}

type fakeOrderStore struct {
	get        func(context.Context, string) (*do.OrderInfoDO, error)
	create     func(context.Context, *gorm.DB, *do.OrderInfoDO) error
	update     func(context.Context, *gorm.DB, *do.OrderInfoDO) error
	listClose  func(context.Context, []string, time.Time, int) ([]*do.OrderInfoDO, error)
	listFinish func(context.Context, string, time.Time, int) ([]*do.OrderInfoDO, error)
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

func (f fakeOrderStore) Create(ctx context.Context, txn *gorm.DB, order *do.OrderInfoDO) error {
	if f.create != nil {
		return f.create(ctx, txn, order)
	}
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

func (f fakeOrderStore) ListCloseCandidates(ctx context.Context, statuses []string, createdBefore time.Time, limit int) ([]*do.OrderInfoDO, error) {
	if f.listClose != nil {
		return f.listClose(ctx, statuses, createdBefore, limit)
	}
	return nil, nil
}

func (f fakeOrderStore) ListFinishCandidates(ctx context.Context, status string, paidBefore time.Time, limit int) ([]*do.OrderInfoDO, error) {
	if f.listFinish != nil {
		return f.listFinish(ctx, status, paidBefore, limit)
	}
	return nil, nil
}

type fakeOrderStatusLogStore struct {
	create        func(context.Context, *gorm.DB, *do.OrderStatusLogDO) error
	listByOrderSn func(context.Context, string) ([]*do.OrderStatusLogDO, error)
}

func (f fakeOrderStatusLogStore) Create(ctx context.Context, txn *gorm.DB, entry *do.OrderStatusLogDO) error {
	if f.create != nil {
		return f.create(ctx, txn, entry)
	}
	return nil
}

func (f fakeOrderStatusLogStore) ListByOrderSn(ctx context.Context, orderSn string) ([]*do.OrderStatusLogDO, error) {
	if f.listByOrderSn != nil {
		return f.listByOrderSn(ctx, orderSn)
	}
	return nil, nil
}

type fakeInventoryGateway struct {
	release func(context.Context, string, []boundary.InventoryItem) error
}

func (f fakeInventoryGateway) Release(ctx context.Context, orderSn string, items []boundary.InventoryItem) error {
	if f.release != nil {
		return f.release(ctx, orderSn, items)
	}
	return nil
}

type fakeGoodsGateway struct {
	batchGetGoods func(context.Context, []int32) (map[int32]boundary.GoodsInfo, error)
}

func (f fakeGoodsGateway) BatchGetGoods(ctx context.Context, ids []int32) (map[int32]boundary.GoodsInfo, error) {
	if f.batchGetGoods != nil {
		return f.batchGetGoods(ctx, ids)
	}
	return map[int32]boundary.GoodsInfo{}, nil
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
