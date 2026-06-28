package db

import (
	"context"
	gpbv1 "goshop/api/goods/v1"
	"goshop/app/pkg/options"
	"goshop/gmicro/registry/consul"
	"goshop/gmicro/server/rpcserver"
	"goshop/gmicro/server/rpcserver/clientinterceptors"

	cosulAPI "github.com/hashicorp/consul/api"

	"goshop/gmicro/registry"
)

const goodsServiceName = "discovery:///goshop-goods-srv"

func NewDiscovery(opts *options.RegistryOptions) registry.Discovery {
	c := cosulAPI.DefaultConfig()
	c.Address = opts.Address
	c.Scheme = opts.Scheme
	cli, err := cosulAPI.NewClient(c)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(true))
	return r
}

func GetGoodsClient(opts *options.RegistryOptions) gpbv1.GoodsClient {
	discovery := NewDiscovery(opts)
	goodsClient := NewGoodsServiceClient(discovery)
	return goodsClient
}

func NewGoodsServiceClient(r registry.Discovery) gpbv1.GoodsClient {
	conn, err := rpcserver.DialInsecure(
		context.Background(),
		rpcserver.WithEndpoint(goodsServiceName),
		rpcserver.WithDiscovery(r),
		rpcserver.WithConnectProbe(true),
		rpcserver.WithClientUnaryInterceptor(clientinterceptors.UnaryTracingInterceptor),
	)
	if err != nil {
		panic(err)
	}
	c := gpbv1.NewGoodsClient(conn)
	return c
}
