package v1

import (
	"context"

	ipb "goshop/api/inventory/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"
)

type InventorySrv interface {
	Detail(ctx context.Context, goodsID uint64) (*ipb.GoodsInvInfo, error)
	OrderDetail(ctx context.Context, orderSn string) (*ipb.SellDetailInfo, error)
}

type inventoryService struct {
	data data.DataFactory
}

func (is *inventoryService) Detail(ctx context.Context, goodsID uint64) (*ipb.GoodsInvInfo, error) {
	if is == nil || is.data == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "inventory data client is not initialized")
	}
	if goodsID == 0 {
		return nil, errors.WithCode(code.ErrInventoryNotFound, "inventory not found")
	}

	client := is.data.Inventory()
	if client == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "inventory grpc client is not initialized")
	}

	resp, err := client.GetStock(ctx, &ipb.GoodsInvInfo{GoodsId: int32(goodsID)})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "inventory grpc response is empty")
	}
	return resp, nil
}

func (is *inventoryService) OrderDetail(ctx context.Context, orderSn string) (*ipb.SellDetailInfo, error) {
	if is == nil || is.data == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "inventory data client is not initialized")
	}
	if orderSn == "" {
		return nil, errors.WithCode(code.ErrInvSellDetailNotFound, "inventory sell detail not found")
	}

	client := is.data.Inventory()
	if client == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "inventory grpc client is not initialized")
	}

	resp, err := client.GetSellDetail(ctx, &ipb.OrderInfo{OrderSn: orderSn})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "inventory grpc response is empty")
	}
	return resp, nil
}

func NewInventory(data data.DataFactory) *inventoryService {
	return &inventoryService{data: data}
}

var _ InventorySrv = &inventoryService{}
