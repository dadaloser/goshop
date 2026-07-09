package v1

import (
	"context"
	"testing"

	datav1 "goshop/app/inventory/srv/internal/data/v1"
	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/inventory/srv/internal/domain/dto"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"gorm.io/gorm"
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

func TestConfirmMarksSellDetailConfirmed(t *testing.T) {
	var gotStatus int32
	srv := &inventoryService{
		pool:    nil,
		testTxn: fakeTxn{},
		data: fakeInventoryDataFactory{
			store: fakeInventoryStore{
				getSellDetail: func(context.Context, *gorm.DB, string) (*do.StockSellDetailDO, error) {
					return &do.StockSellDetailDO{
						OrderSn: "order-1",
						Status:  stockSellStatusReserved,
						Detail:  do.GoodsDetailList{{Goods: 1, Num: 1}},
					}, nil
				},
				updateSellDetailStatus: func(_ context.Context, _ *gorm.DB, _ string, status int32) error {
					gotStatus = status
					return nil
				},
			},
		},
	}

	err := srv.Confirm(context.Background(), "order-1", []do.GoodsDetail{{Goods: 1, Num: 1}})
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if gotStatus != stockSellStatusConfirmed {
		t.Fatalf("Confirm() status = %d, want %d", gotStatus, stockSellStatusConfirmed)
	}
}

func TestConfirmIsIdempotentForConfirmedSellDetail(t *testing.T) {
	var updateCalled bool
	srv := &inventoryService{
		pool:    nil,
		testTxn: fakeTxn{},
		data: fakeInventoryDataFactory{
			store: fakeInventoryStore{
				getSellDetail: func(context.Context, *gorm.DB, string) (*do.StockSellDetailDO, error) {
					return &do.StockSellDetailDO{
						OrderSn: "order-1",
						Status:  stockSellStatusConfirmed,
						Detail:  do.GoodsDetailList{{Goods: 1, Num: 1}},
					}, nil
				},
				updateSellDetailStatus: func(_ context.Context, _ *gorm.DB, _ string, status int32) error {
					updateCalled = true
					return nil
				},
			},
		},
	}

	err := srv.Confirm(context.Background(), "order-1", []do.GoodsDetail{{Goods: 1, Num: 1}})
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if updateCalled {
		t.Fatal("Confirm() updated status for already confirmed detail")
	}
}

func TestReleaseIgnoresConfirmedSellDetail(t *testing.T) {
	var increaseCalled bool
	srv := &inventoryService{
		pool:    nil,
		testTxn: fakeTxn{},
		data: fakeInventoryDataFactory{
			store: fakeInventoryStore{
				getSellDetail: func(context.Context, *gorm.DB, string) (*do.StockSellDetailDO, error) {
					return &do.StockSellDetailDO{
						OrderSn: "order-1",
						Status:  stockSellStatusConfirmed,
						Detail:  do.GoodsDetailList{{Goods: 1, Num: 1}},
					}, nil
				},
				increase: func(context.Context, *gorm.DB, uint64, int) error {
					increaseCalled = true
					return nil
				},
			},
		},
	}

	err := srv.Release(context.Background(), "order-1", []do.GoodsDetail{{Goods: 1, Num: 1}})
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if increaseCalled {
		t.Fatal("Release() increased stock for confirmed detail")
	}
}

type fakeInventoryDataFactory struct {
	store datav1.InventoryStore
	txn   txExecutor
}

func (f fakeInventoryDataFactory) Inventories() datav1.InventoryStore {
	return f.store
}

func (f fakeInventoryDataFactory) Begin() *gorm.DB {
	return &gorm.DB{}
}

type fakeTxn struct{}

func (fakeTxn) DB() *gorm.DB {
	return &gorm.DB{}
}

func (fakeTxn) Commit() error {
	return nil
}

func (fakeTxn) Rollback() error {
	return nil
}

type fakeInventoryStore struct {
	create                 func(context.Context, *do.InventoryDO) error
	get                    func(context.Context, uint64) (*do.InventoryDO, error)
	getSellDetail          func(context.Context, *gorm.DB, string) (*do.StockSellDetailDO, error)
	reduce                 func(context.Context, *gorm.DB, uint64, int) error
	increase               func(context.Context, *gorm.DB, uint64, int) error
	createStockSellDetail  func(context.Context, *gorm.DB, *do.StockSellDetailDO) error
	createStockIfAbsent    func(context.Context, *gorm.DB, *do.StockSellDetailDO) (bool, error)
	updateSellDetailStatus func(context.Context, *gorm.DB, string, int32) error
}

func (f fakeInventoryStore) Create(ctx context.Context, inv *do.InventoryDO) error {
	if f.create != nil {
		return f.create(ctx, inv)
	}
	return nil
}

func (f fakeInventoryStore) Get(ctx context.Context, goodsID uint64) (*do.InventoryDO, error) {
	if f.get != nil {
		return f.get(ctx, goodsID)
	}
	return nil, errors.WithCode(code.ErrInventoryNotFound, "inventory not found")
}

func (f fakeInventoryStore) GetSellDetail(ctx context.Context, txn *gorm.DB, ordersn string) (*do.StockSellDetailDO, error) {
	if f.getSellDetail != nil {
		return f.getSellDetail(ctx, txn, ordersn)
	}
	return nil, errors.WithCode(code.ErrInvSellDetailNotFound, "inventory sell detail not found")
}

func (f fakeInventoryStore) Reduce(ctx context.Context, txn *gorm.DB, goodsID uint64, num int) error {
	if f.reduce != nil {
		return f.reduce(ctx, txn, goodsID, num)
	}
	return nil
}

func (f fakeInventoryStore) Increase(ctx context.Context, txn *gorm.DB, goodsID uint64, num int) error {
	if f.increase != nil {
		return f.increase(ctx, txn, goodsID, num)
	}
	return nil
}

func (f fakeInventoryStore) CreateStockSellDetail(ctx context.Context, txn *gorm.DB, detail *do.StockSellDetailDO) error {
	if f.createStockSellDetail != nil {
		return f.createStockSellDetail(ctx, txn, detail)
	}
	return nil
}

func (f fakeInventoryStore) CreateStockSellDetailIfAbsent(ctx context.Context, txn *gorm.DB, detail *do.StockSellDetailDO) (bool, error) {
	if f.createStockIfAbsent != nil {
		return f.createStockIfAbsent(ctx, txn, detail)
	}
	return false, nil
}

func (f fakeInventoryStore) UpdateStockSellDetailStatus(ctx context.Context, txn *gorm.DB, ordersn string, status int32) error {
	if f.updateSellDetailStatus != nil {
		return f.updateSellDetailStatus(ctx, txn, ordersn, status)
	}
	return nil
}
