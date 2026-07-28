package v1

import (
	"context"
	"testing"

	invpb "goshop/api/inventory/v1"
	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/inventory/srv/internal/domain/dto"
	svcv1 "goshop/app/inventory/srv/internal/service/v1"
	"goshop/gmicro/errcode"
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
		{
			name: "get sell detail",
			run: func() error {
				_, err := server.GetSellDetail(context.Background(), nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, errcode.ErrValidation) {
				t.Fatalf("error = %v, want code %d", err, errcode.ErrValidation)
			}
		})
	}
}

func TestInventoryServerGetSellDetailReturnsStatusAndItems(t *testing.T) {
	server := &inventoryServer{
		srv: fakeServiceFactory{
			inventory: fakeInventoryService{
				getOrderFlow: func(context.Context, string) (*do.OrderFlow, error) {
					return &do.OrderFlow{
						SellDetail: &do.StockSellDetailDO{
							OrderSn: "order-1",
							Status:  3,
							Detail: do.GoodsDetailList{
								{Goods: 11, Num: 2},
								{Goods: 12, Num: 1},
							},
						},
						Adjustments: []do.InventoryAdjustmentDO{
							{GoodsID: 11, BeforeAvailable: 7, AfterAvailable: 5, CorrelationID: "corr-1"},
						},
						Inventories: []do.InventoryDO{
							{Goods: 11, Stocks: 5, Total: 10, Available: 5, Locked: 3, Sold: 2},
						},
					}, nil
				},
			},
		},
	}

	resp, err := server.GetSellDetail(context.Background(), &invpb.OrderInfo{OrderSn: "order-1"})
	if err != nil {
		t.Fatalf("GetSellDetail() error = %v", err)
	}
	if resp.OrderSn != "order-1" || resp.Status != 3 || resp.StatusName != "confirmed" {
		t.Fatalf("GetSellDetail() = %+v", resp)
	}
	if len(resp.GoodsInfo) != 2 || resp.GoodsInfo[0].GoodsId != 11 || resp.GoodsInfo[0].Num != 2 {
		t.Fatalf("GetSellDetail() goods = %+v", resp.GoodsInfo)
	}
	if len(resp.Adjustments) != 1 || resp.Adjustments[0].CorrelationId != "corr-1" {
		t.Fatalf("GetSellDetail() adjustments = %+v", resp.Adjustments)
	}
	if len(resp.InventorySnapshot) != 1 || resp.InventorySnapshot[0].GoodsId != 11 || resp.InventorySnapshot[0].Available != 5 {
		t.Fatalf("GetSellDetail() inventory snapshot = %+v", resp.InventorySnapshot)
	}
}

func TestInventoryServerInvDetailReturnsLifecycleFields(t *testing.T) {
	server := &inventoryServer{
		srv: fakeServiceFactory{
			inventory: fakeInventoryService{
				get: func(context.Context, uint64) (*dto.InventoryDTO, error) {
					return &dto.InventoryDTO{InventoryDO: do.InventoryDO{
						Goods:     11,
						Stocks:    7,
						Total:     10,
						Available: 7,
						Locked:    2,
						Sold:      1,
					}}, nil
				},
			},
		},
	}

	resp, err := server.InvDetail(context.Background(), &invpb.GoodsInvInfo{GoodsId: 11})
	if err != nil {
		t.Fatalf("InvDetail() error = %v", err)
	}
	if resp.GoodsId != 11 || resp.Num != 7 || resp.Total != 10 || resp.Available != 7 || resp.Locked != 2 || resp.Sold != 1 {
		t.Fatalf("InvDetail() = %+v, want goods=11 num=7 total=10 available=7 locked=2 sold=1", resp)
	}
}

func TestInventoryServerSetInvPassesLifecycleFields(t *testing.T) {
	var created *dto.InventoryDTO
	server := &inventoryServer{
		srv: fakeServiceFactory{
			inventory: fakeInventoryService{
				create: func(_ context.Context, inv *dto.InventoryDTO) error {
					copied := *inv
					created = &copied
					return nil
				},
			},
		},
	}

	_, err := server.SetInv(context.Background(), &invpb.GoodsInvInfo{
		GoodsId:   22,
		Num:       6,
		Total:     9,
		Available: 6,
		Locked:    2,
		Sold:      1,
	})
	if err != nil {
		t.Fatalf("SetInv() error = %v", err)
	}
	if created == nil {
		t.Fatal("SetInv() did not call create")
	}
	if created.Goods != 22 || created.Stocks != 6 || created.Total != 9 || created.Available != 6 || created.Locked != 2 || created.Sold != 1 {
		t.Fatalf("SetInv() created = %+v, want goods=22 stocks=6 total=9 available=6 locked=2 sold=1", created)
	}
}

type fakeServiceFactory struct {
	inventory svcv1.InventorySrv
}

func (f fakeServiceFactory) Inventory() svcv1.InventorySrv {
	return f.inventory
}

type fakeInventoryService struct {
	create         func(context.Context, *dto.InventoryDTO) error
	get            func(context.Context, uint64) (*dto.InventoryDTO, error)
	getOrderDetail func(context.Context, string) (*do.StockSellDetailDO, error)
	getOrderFlow   func(context.Context, string) (*do.OrderFlow, error)
	sell           func(context.Context, string, []do.GoodsDetail) error
	reback         func(context.Context, string, []do.GoodsDetail) error
	confirm        func(context.Context, string, []do.GoodsDetail) error
	release        func(context.Context, string, []do.GoodsDetail) error
}

func (f fakeInventoryService) Create(ctx context.Context, inv *dto.InventoryDTO) error {
	if f.create != nil {
		return f.create(ctx, inv)
	}
	return nil
}
func (f fakeInventoryService) Adjust(context.Context, *dto.InventoryDTO, *do.InventoryAdjustmentDO) error {
	return nil
}
func (f fakeInventoryService) ListAdjustments(context.Context, uint64, int, int) ([]do.InventoryAdjustmentDO, int64, error) {
	return []do.InventoryAdjustmentDO{}, 0, nil
}

func (f fakeInventoryService) Get(ctx context.Context, goodsID uint64) (*dto.InventoryDTO, error) {
	if f.get != nil {
		return f.get(ctx, goodsID)
	}
	return nil, nil
}

func (f fakeInventoryService) GetOrderDetail(ctx context.Context, orderSn string) (*do.StockSellDetailDO, error) {
	if f.getOrderDetail != nil {
		return f.getOrderDetail(ctx, orderSn)
	}
	return nil, nil
}

func (f fakeInventoryService) GetOrderFlow(ctx context.Context, orderSn string) (*do.OrderFlow, error) {
	if f.getOrderFlow != nil {
		return f.getOrderFlow(ctx, orderSn)
	}
	if f.getOrderDetail != nil {
		detail, err := f.getOrderDetail(ctx, orderSn)
		if err != nil {
			return nil, err
		}
		return &do.OrderFlow{SellDetail: detail}, nil
	}
	return nil, nil
}

func (f fakeInventoryService) Sell(ctx context.Context, orderSn string, detail []do.GoodsDetail) error {
	if f.sell != nil {
		return f.sell(ctx, orderSn, detail)
	}
	return nil
}

func (f fakeInventoryService) Reback(ctx context.Context, orderSn string, detail []do.GoodsDetail) error {
	if f.reback != nil {
		return f.reback(ctx, orderSn, detail)
	}
	return nil
}

func (f fakeInventoryService) Confirm(ctx context.Context, orderSn string, detail []do.GoodsDetail) error {
	if f.confirm != nil {
		return f.confirm(ctx, orderSn, detail)
	}
	return nil
}

func (f fakeInventoryService) Release(ctx context.Context, orderSn string, detail []do.GoodsDetail) error {
	if f.release != nil {
		return f.release(ctx, orderSn, detail)
	}
	return nil
}
