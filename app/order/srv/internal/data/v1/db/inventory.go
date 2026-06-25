package db

import (
	"context"

	proto "goshop/api/inventory/v1"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/rpcserver"
	"goshop/gmicro/server/rpcserver/clientinterceptors"

	"goshop/gmicro/registry"
)

const ginvserviceName = "discovery:///goshop-inventory-srv"

func GetInventoryClient(opts *options.RegistryOptions) proto.InventoryClient {
	discovery := NewDiscovery(opts)
	invClient := NewInventoryServiceClient(discovery)
	return invClient
}

func NewInventoryServiceClient(r registry.Discovery) proto.InventoryClient {
	conn, err := rpcserver.DialInsecure(
		context.Background(),
		rpcserver.WithEndpoint(ginvserviceName),
		rpcserver.WithDiscovery(r),
		rpcserver.WithClientUnaryInterceptor(clientinterceptors.UnaryTracingInterceptor),
	)
	if err != nil {
		panic(err)
	}
	c := proto.NewInventoryClient(conn)
	return c
}
