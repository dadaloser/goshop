package boundary

import (
	"context"
	ipb "goshop/api/inventory/v1"
	"goshop/app/pkg/client"
	"goshop/app/pkg/options"
	"goshop/gmicro/resilience"
	"goshop/gmicro/server/rpcserver"
)

type InventoryItem struct {
	GoodsID int32
	Num     int32
}

type InventoryGateway interface {
	Release(ctx context.Context, orderSn string, items []InventoryItem) error
}

type inventoryRPCGateway struct {
	client ipb.InventoryClient
}

func NewInventoryRPCGatewayContext(
	ctx context.Context,
	registry *options.RegistryOptions,
	rpcSecurity *rpcserver.SecurityPolicy,
	rpcResilience *resilience.Options,
) (InventoryGateway, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	inventoryClient, _, err := client.NewInventoryClient(
		ctx,
		registry,
		rpcSecurity,
		rpcserver.WithClientResilience(rpcResilience),
	)
	if err != nil {
		return nil, err
	}
	return &inventoryRPCGateway{client: inventoryClient}, nil
}

func (g *inventoryRPCGateway) Release(ctx context.Context, orderSn string, items []InventoryItem) error {
	req := &ipb.SellInfo{OrderSn: orderSn}
	for _, item := range items {
		req.GoodsInfo = append(req.GoodsInfo, &ipb.GoodsInvInfo{
			GoodsId: item.GoodsID,
			Num:     item.Num,
		})
	}
	_, err := g.client.Release(ctx, req)
	return err
}

var _ InventoryGateway = &inventoryRPCGateway{}
