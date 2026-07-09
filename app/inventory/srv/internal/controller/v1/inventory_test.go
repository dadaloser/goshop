package v1

import (
	"context"
	"testing"

	invpb "goshop/api/inventory/v1"
	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/inventory/srv/internal/domain/dto"
	svcv1 "goshop/app/inventory/srv/internal/service/v1"
	code2 "goshop/gmicro/code"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, code2.ErrValidation) {
				t.Fatalf("error = %v, want code %d", err, code2.ErrValidation)
			}
		})
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
	create  func(context.Context, *dto.InventoryDTO) error
	get     func(context.Context, uint64) (*dto.InventoryDTO, error)
	sell    func(context.Context, string, []do.GoodsDetail) error
	reback  func(context.Context, string, []do.GoodsDetail) error
	confirm func(context.Context, string, []do.GoodsDetail) error
	release func(context.Context, string, []do.GoodsDetail) error
}

func (f fakeInventoryService) Create(ctx context.Context, inv *dto.InventoryDTO) error {
	if f.create != nil {
		return f.create(ctx, inv)
	}
	return nil
}

func (f fakeInventoryService) Get(ctx context.Context, goodsID uint64) (*dto.InventoryDTO, error) {
	if f.get != nil {
		return f.get(ctx, goodsID)
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
