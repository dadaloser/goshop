package boundary

import (
	"context"
	goodspb "goshop/api/goods/v1"
	"goshop/app/pkg/client"
	"goshop/app/pkg/options"
)

type GoodsInfo struct {
	ID              int32
	Name            string
	ShopPrice       float32
	GoodsFrontImage string
}

type GoodsGateway interface {
	BatchGetGoods(ctx context.Context, ids []int32) (map[int32]GoodsInfo, error)
}

type goodsRPCGateway struct {
	client goodspb.GoodsClient
}

// NewGoodsRPCGatewayContext creates a goods gateway using ctx for the initial
// gRPC dial and discovery probe.
func NewGoodsRPCGatewayContext(ctx context.Context, registry *options.RegistryOptions, rpcSecurity *options.RPCSecurityOptions) (GoodsGateway, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	goodsClient, _, err := client.NewGoodsClient(ctx, registry, rpcSecurity)
	if err != nil {
		return nil, err
	}
	return &goodsRPCGateway{client: goodsClient}, nil
}

func (g *goodsRPCGateway) BatchGetGoods(ctx context.Context, ids []int32) (map[int32]GoodsInfo, error) {
	resp, err := g.client.BatchGetGoods(ctx, &goodspb.BatchGoodsIdInfo{Id: ids})
	if err != nil {
		return nil, err
	}

	goods := make(map[int32]GoodsInfo, len(resp.Data))
	for _, item := range resp.Data {
		goods[item.Id] = GoodsInfo{
			ID:              item.Id,
			Name:            item.Name,
			ShopPrice:       item.ShopPrice,
			GoodsFrontImage: item.GoodsFrontImage,
		}
	}
	return goods, nil
}

var _ GoodsGateway = &goodsRPCGateway{}
