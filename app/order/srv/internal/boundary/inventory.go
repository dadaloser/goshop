package boundary

import (
	"context"
	"errors"
	ipb "goshop/api/inventory/v1"
	"goshop/app/pkg/client"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/rpcserver"
	"goshop/pkg/resilience"
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
	rpcSecurity *options.RPCSecurityOptions,
	rpcResilience *resilience.Options,
) (InventoryGateway, error) {
	if ctx == nil {
		return nil, errors.New("inventory RPC gateway requires a startup context")
	}
	dialCtx, cancel := context.WithTimeout(ctx, upstreamDialTimeout)
	defer cancel()
	inventoryClient, _, err := client.NewInventoryClient(
		dialCtx,
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
