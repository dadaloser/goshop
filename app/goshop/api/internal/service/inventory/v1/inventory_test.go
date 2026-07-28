package v1

import (
	"context"
	"goshop/app/pkg/bizcode"
	"testing"

	gpb "goshop/api/goods/v1"
	ipb "goshop/api/inventory/v1"
	opb "goshop/api/order/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/pkg/errors"

	"google.golang.org/grpc"
)

func TestInventoryServiceDetailRejectsInvalidBoundary(t *testing.T) {
	tests := []struct {
		name string
		svc  InventorySrv
		id   uint64
		code int
		kind errors.Kind
	}{
		{name: "nil service", svc: (*inventoryService)(nil), id: 1, code: bizcode.ErrConnectGRPC, kind: errors.KindUnavailable},
		{name: "nil data factory", svc: NewInventory(nil), id: 1, code: bizcode.ErrConnectGRPC, kind: errors.KindUnavailable},
		{name: "zero goods id", svc: NewInventory(&fakeInventoryDataFactory{inventory: &fakeInventoryClient{}}), id: 0, code: bizcode.ErrInventoryNotFound, kind: errors.KindNotFound},
		{name: "nil inventory client", svc: NewInventory(&fakeInventoryDataFactory{}), id: 1, code: bizcode.ErrConnectGRPC, kind: errors.KindUnavailable},
		{name: "nil downstream response", svc: NewInventory(&fakeInventoryDataFactory{inventory: &fakeInventoryClient{}}), id: 1, code: bizcode.ErrConnectGRPC, kind: errors.KindUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc.Detail(context.Background(), tt.id)
			if !errors.IsCode(err, tt.code) {
				t.Fatalf("Detail() error = %v, want code %d", err, tt.code)
			}
			spec, ok := errors.SpecOf(err)
			if !ok {
				t.Fatal("Detail() error has no specification")
			}
			if spec.Kind != tt.kind {
				t.Fatalf("Detail() error kind = %q, want %q", spec.Kind, tt.kind)
			}
		})
	}
}

func TestInventoryServiceOrderDetailRejectsEmptyOrderSN(t *testing.T) {
	svc := NewInventory(&fakeInventoryDataFactory{inventory: &fakeInventoryClient{}})
	_, err := svc.OrderDetail(context.Background(), "")

	if !errors.IsCode(err, bizcode.ErrInvSellDetailNotFound) {
		t.Fatalf("OrderDetail() error = %v, want code %d", err, bizcode.ErrInvSellDetailNotFound)
	}
	spec, ok := errors.SpecOf(err)
	if !ok {
		t.Fatal("OrderDetail() error has no specification")
	}
	if spec.Kind != errors.KindInvalidArgument {
		t.Fatalf("OrderDetail() error kind = %q, want %q", spec.Kind, errors.KindInvalidArgument)
	}
}

func TestInventoryServiceDetailCallsInventoryClient(t *testing.T) {
	client := &fakeInventoryClient{
		getStockResp: &ipb.GoodsInvInfo{
			GoodsId:   5,
			Num:       7,
			Total:     10,
			Available: 7,
			Locked:    2,
			Sold:      1,
		},
	}
	svc := NewInventory(&fakeInventoryDataFactory{inventory: client})

	resp, err := svc.Detail(context.Background(), 5)
	if err != nil {
		t.Fatalf("Detail() error = %v", err)
	}
	if resp.GetGoodsId() != 5 || resp.GetAvailable() != 7 || resp.GetLocked() != 2 || resp.GetSold() != 1 {
		t.Fatalf("Detail() response = %+v", resp)
	}
	if client.gotGetStock == nil || client.gotGetStock.GetGoodsId() != 5 {
		t.Fatalf("GetStock() request = %+v, want goods_id=5", client.gotGetStock)
	}
}

func TestInventoryServiceOrderDetail(t *testing.T) {
	client := &fakeInventoryClient{
		getSellDetailResp: &ipb.SellDetailInfo{
			OrderSn:    "order-1",
			Status:     3,
			StatusName: "confirmed",
			GoodsInfo: []*ipb.GoodsInvInfo{
				{GoodsId: 5, Num: 2},
			},
		},
	}
	svc := NewInventory(&fakeInventoryDataFactory{inventory: client})

	resp, err := svc.OrderDetail(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("OrderDetail() error = %v", err)
	}
	if resp.GetOrderSn() != "order-1" || resp.GetStatusName() != "confirmed" {
		t.Fatalf("OrderDetail() response = %+v", resp)
	}
	if client.gotGetSellDetail == nil || client.gotGetSellDetail.GetOrderSn() != "order-1" {
		t.Fatalf("GetSellDetail() request = %+v, want order-1", client.gotGetSellDetail)
	}
}

type fakeInventoryDataFactory struct {
	inventory *fakeInventoryClient
}

func (f *fakeInventoryDataFactory) Goods() gpb.GoodsClient {
	return nil
}

func (f *fakeInventoryDataFactory) Orders() opb.OrderClient {
	return nil
}

func (f *fakeInventoryDataFactory) Inventory() ipb.InventoryClient {
	if f.inventory == nil {
		return nil
	}
	return f.inventory
}

func (f *fakeInventoryDataFactory) Users() data.UserData {
	return nil
}

type fakeInventoryClient struct {
	ipb.InventoryClient

	getStockResp      *ipb.GoodsInvInfo
	getStockErr       error
	gotGetStock       *ipb.GoodsInvInfo
	getSellDetailResp *ipb.SellDetailInfo
	getSellDetailErr  error
	gotGetSellDetail  *ipb.OrderInfo
}

func (f *fakeInventoryClient) GetStock(ctx context.Context, in *ipb.GoodsInvInfo, opts ...grpc.CallOption) (*ipb.GoodsInvInfo, error) {
	f.gotGetStock = in
	return f.getStockResp, f.getStockErr
}

func (f *fakeInventoryClient) GetSellDetail(ctx context.Context, in *ipb.OrderInfo, opts ...grpc.CallOption) (*ipb.SellDetailInfo, error) {
	f.gotGetSellDetail = in
	return f.getSellDetailResp, f.getSellDetailErr
}
