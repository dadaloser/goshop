package v1

import (
	"context"
	"reflect"
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
		{
			name: "get order detail empty order sn",
			run: func() error {
				_, err := srv.GetOrderDetail(context.Background(), " ")
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

func TestInventoryServiceGetOrderDetailDelegatesToStore(t *testing.T) {
	srv := &inventoryService{
		data: fakeInventoryDataFactory{
			store: fakeInventoryStore{
				getSellDetail: func(context.Context, *gorm.DB, string) (*do.StockSellDetailDO, error) {
					return &do.StockSellDetailDO{
						OrderSn: "order-1",
						Status:  stockSellStatusReserved,
						Detail:  do.GoodsDetailList{{Goods: 1, Num: 2}},
					}, nil
				},
			},
		},
	}

	detail, err := srv.GetOrderDetail(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("GetOrderDetail() error = %v", err)
	}
	if detail == nil || detail.OrderSn != "order-1" || detail.Status != stockSellStatusReserved {
		t.Fatalf("GetOrderDetail() = %+v", detail)
	}
}

func TestConfirmMarksSellDetailConfirmed(t *testing.T) {
	var gotStatus int32
	var confirmed []do.GoodsDetail
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
				confirmSell: func(_ context.Context, _ *gorm.DB, goodsID uint64, num int) error {
					confirmed = append(confirmed, do.GoodsDetail{Goods: int32(goodsID), Num: int32(num)})
					return nil
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
	if !reflect.DeepEqual(confirmed, []do.GoodsDetail{{Goods: 1, Num: 1}}) {
		t.Fatalf("Confirm() confirmed detail = %+v, want %+v", confirmed, []do.GoodsDetail{{Goods: 1, Num: 1}})
	}
	if gotStatus != stockSellStatusConfirmed {
		t.Fatalf("Confirm() status = %d, want %d", gotStatus, stockSellStatusConfirmed)
	}
}

func TestConfirmIsIdempotentForConfirmedSellDetail(t *testing.T) {
	var updateCalled bool
	var confirmCalled bool
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
				confirmSell: func(_ context.Context, _ *gorm.DB, _ uint64, _ int) error {
					confirmCalled = true
					return nil
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
	if confirmCalled {
		t.Fatal("Confirm() confirmed stock for already confirmed detail")
	}
	if updateCalled {
		t.Fatal("Confirm() updated status for already confirmed detail")
	}
}

func TestConfirmUsesPersistedDetailSorted(t *testing.T) {
	var confirmed []do.GoodsDetail
	srv := &inventoryService{
		pool:    nil,
		testTxn: fakeTxn{},
		data: fakeInventoryDataFactory{
			store: fakeInventoryStore{
				getSellDetail: func(context.Context, *gorm.DB, string) (*do.StockSellDetailDO, error) {
					return &do.StockSellDetailDO{
						OrderSn: "order-1",
						Status:  stockSellStatusReserved,
						Detail: do.GoodsDetailList{
							{Goods: 3, Num: 1},
							{Goods: 1, Num: 2},
							{Goods: 2, Num: 1},
						},
					}, nil
				},
				confirmSell: func(_ context.Context, _ *gorm.DB, goodsID uint64, num int) error {
					confirmed = append(confirmed, do.GoodsDetail{Goods: int32(goodsID), Num: int32(num)})
					return nil
				},
			},
		},
	}

	err := srv.Confirm(context.Background(), "order-1", []do.GoodsDetail{{Goods: 99, Num: 99}})
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}

	want := []do.GoodsDetail{
		{Goods: 1, Num: 2},
		{Goods: 2, Num: 1},
		{Goods: 3, Num: 1},
	}
	if !reflect.DeepEqual(confirmed, want) {
		t.Fatalf("Confirm() confirmed detail = %+v, want %+v", confirmed, want)
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

func TestSellIsIdempotentForReservedSellDetail(t *testing.T) {
	var reduceCalled bool
	var createCalled bool
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
				reduce: func(context.Context, *gorm.DB, uint64, int) error {
					reduceCalled = true
					return nil
				},
				createStockSellDetail: func(context.Context, *gorm.DB, *do.StockSellDetailDO) error {
					createCalled = true
					return nil
				},
			},
		},
	}

	err := srv.Sell(context.Background(), "order-1", []do.GoodsDetail{{Goods: 1, Num: 1}})
	if err != nil {
		t.Fatalf("Sell() error = %v", err)
	}
	if reduceCalled {
		t.Fatal("Sell() reduced inventory for already reserved order")
	}
	if createCalled {
		t.Fatal("Sell() created sell detail for already reserved order")
	}
}

func TestSellIgnoresDelayedRequestAfterReleaseMarker(t *testing.T) {
	var reduceCalled bool
	var createCalled bool
	srv := &inventoryService{
		pool:    nil,
		testTxn: fakeTxn{},
		data: fakeInventoryDataFactory{
			store: fakeInventoryStore{
				getSellDetail: func(context.Context, *gorm.DB, string) (*do.StockSellDetailDO, error) {
					return &do.StockSellDetailDO{
						OrderSn: "order-1",
						Status:  stockSellStatusReleased,
						Detail:  do.GoodsDetailList{{Goods: 1, Num: 1}},
					}, nil
				},
				reduce: func(context.Context, *gorm.DB, uint64, int) error {
					reduceCalled = true
					return nil
				},
				createStockSellDetail: func(context.Context, *gorm.DB, *do.StockSellDetailDO) error {
					createCalled = true
					return nil
				},
			},
		},
	}

	err := srv.Sell(context.Background(), "order-1", []do.GoodsDetail{{Goods: 1, Num: 1}})
	if err != nil {
		t.Fatalf("Sell() error = %v", err)
	}
	if reduceCalled {
		t.Fatal("Sell() reduced inventory after release marker")
	}
	if createCalled {
		t.Fatal("Sell() created sell detail after release marker")
	}
}

func TestSellSortsDetailBeforeReduceAndPersist(t *testing.T) {
	var reducedGoods []do.GoodsDetail
	var createdDetail do.GoodsDetailList
	srv := &inventoryService{
		pool:    nil,
		testTxn: fakeTxn{},
		data: fakeInventoryDataFactory{
			store: fakeInventoryStore{
				getSellDetail: func(context.Context, *gorm.DB, string) (*do.StockSellDetailDO, error) {
					return nil, errors.WithCode(code.ErrInvSellDetailNotFound, "inventory sell detail not found")
				},
				reduce: func(_ context.Context, _ *gorm.DB, goodsID uint64, num int) error {
					reducedGoods = append(reducedGoods, do.GoodsDetail{Goods: int32(goodsID), Num: int32(num)})
					return nil
				},
				createStockSellDetail: func(_ context.Context, _ *gorm.DB, detail *do.StockSellDetailDO) error {
					createdDetail = append(createdDetail, detail.Detail...)
					return nil
				},
			},
		},
	}

	err := srv.Sell(context.Background(), "order-1", []do.GoodsDetail{
		{Goods: 3, Num: 1},
		{Goods: 1, Num: 2},
		{Goods: 2, Num: 1},
	})
	if err != nil {
		t.Fatalf("Sell() error = %v", err)
	}

	want := []do.GoodsDetail{
		{Goods: 1, Num: 2},
		{Goods: 2, Num: 1},
		{Goods: 3, Num: 1},
	}
	if !reflect.DeepEqual(reducedGoods, want) {
		t.Fatalf("Sell() reduce order = %+v, want %+v", reducedGoods, want)
	}
	if !reflect.DeepEqual([]do.GoodsDetail(createdDetail), want) {
		t.Fatalf("Sell() persisted detail = %+v, want %+v", createdDetail, want)
	}
}

func TestReleaseWritesCancelMarkerWhenSellDetailMissing(t *testing.T) {
	var createdDetail *do.StockSellDetailDO
	srv := &inventoryService{
		pool:    nil,
		testTxn: fakeTxn{},
		data: fakeInventoryDataFactory{
			store: fakeInventoryStore{
				getSellDetail: func(context.Context, *gorm.DB, string) (*do.StockSellDetailDO, error) {
					return nil, errors.WithCode(code.ErrInvSellDetailNotFound, "inventory sell detail not found")
				},
				createStockIfAbsent: func(_ context.Context, _ *gorm.DB, detail *do.StockSellDetailDO) (bool, error) {
					copied := *detail
					copied.Detail = append(do.GoodsDetailList(nil), detail.Detail...)
					createdDetail = &copied
					return true, nil
				},
			},
		},
	}

	err := srv.Release(context.Background(), "order-1", []do.GoodsDetail{
		{Goods: 3, Num: 1},
		{Goods: 1, Num: 2},
	})
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if createdDetail == nil {
		t.Fatal("Release() did not create cancel marker")
	}
	if createdDetail.Status != stockSellStatusReleased {
		t.Fatalf("Release() marker status = %d, want %d", createdDetail.Status, stockSellStatusReleased)
	}
	want := []do.GoodsDetail{{Goods: 1, Num: 2}, {Goods: 3, Num: 1}}
	if !reflect.DeepEqual([]do.GoodsDetail(createdDetail.Detail), want) {
		t.Fatalf("Release() marker detail = %+v, want %+v", createdDetail.Detail, want)
	}
}

func TestReleaseIsIdempotentForReleasedSellDetail(t *testing.T) {
	var increaseCalled bool
	var updateCalled bool
	srv := &inventoryService{
		pool:    nil,
		testTxn: fakeTxn{},
		data: fakeInventoryDataFactory{
			store: fakeInventoryStore{
				getSellDetail: func(context.Context, *gorm.DB, string) (*do.StockSellDetailDO, error) {
					return &do.StockSellDetailDO{
						OrderSn: "order-1",
						Status:  stockSellStatusReleased,
						Detail:  do.GoodsDetailList{{Goods: 1, Num: 1}},
					}, nil
				},
				increase: func(context.Context, *gorm.DB, uint64, int) error {
					increaseCalled = true
					return nil
				},
				updateSellDetailStatus: func(context.Context, *gorm.DB, string, int32) error {
					updateCalled = true
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
		t.Fatal("Release() increased stock for already released detail")
	}
	if updateCalled {
		t.Fatal("Release() updated status for already released detail")
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
	confirmSell            func(context.Context, *gorm.DB, uint64, int) error
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
func (f fakeInventoryStore) Adjust(context.Context, *do.InventoryDO, *do.InventoryAdjustmentDO) error {
	return nil
}
func (f fakeInventoryStore) ListAdjustments(context.Context, uint64, int, int) ([]do.InventoryAdjustmentDO, int64, error) {
	return []do.InventoryAdjustmentDO{}, 0, nil
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

func (f fakeInventoryStore) ConfirmSell(ctx context.Context, txn *gorm.DB, goodsID uint64, num int) error {
	if f.confirmSell != nil {
		return f.confirmSell(ctx, txn, goodsID, num)
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
