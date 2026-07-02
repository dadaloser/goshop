package client

import (
	"context"
	"fmt"
	goodspb "goshop/api/goods/v1"
	inventorypb "goshop/api/inventory/v1"
	userpb "goshop/api/user/v1"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/rpcserver"
	"goshop/gmicro/server/rpcserver/clientinterceptors"

	"google.golang.org/grpc"
)

func DialServiceInsecure(
	ctx context.Context,
	registry *options.RegistryOptions,
	service string,
	opts ...rpcserver.ClientOption,
) (*grpc.ClientConn, error) {
	discovery, err := NewConsulDiscovery(registry)
	if err != nil {
		return nil, fmt.Errorf("create discovery for %s: %w", service, err)
	}

	dialOpts := []rpcserver.ClientOption{
		rpcserver.WithEndpoint(ServiceEndpoint(service)),
		rpcserver.WithDiscovery(discovery),
		rpcserver.WithClientUnaryInterceptor(clientinterceptors.UnaryTracingInterceptor),
	}
	dialOpts = append(dialOpts, opts...)
	conn, err := rpcserver.DialDiscoveryInsecure(ctx, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s (%s): %w", service, ServiceEndpoint(service), err)
	}
	return conn, nil
}

func NewGoodsClient(
	ctx context.Context,
	registry *options.RegistryOptions,
	opts ...rpcserver.ClientOption,
) (goodspb.GoodsClient, *grpc.ClientConn, error) {
	conn, err := DialServiceInsecure(ctx, registry, ServiceGoods, opts...)
	if err != nil {
		return nil, nil, err
	}
	return goodspb.NewGoodsClient(conn), conn, nil
}

func NewInventoryClient(
	ctx context.Context,
	registry *options.RegistryOptions,
	opts ...rpcserver.ClientOption,
) (inventorypb.InventoryClient, *grpc.ClientConn, error) {
	conn, err := DialServiceInsecure(ctx, registry, ServiceInventory, opts...)
	if err != nil {
		return nil, nil, err
	}
	return inventorypb.NewInventoryClient(conn), conn, nil
}

func NewUserClient(
	ctx context.Context,
	registry *options.RegistryOptions,
	opts ...rpcserver.ClientOption,
) (userpb.UserClient, *grpc.ClientConn, error) {
	conn, err := DialServiceInsecure(ctx, registry, ServiceUser, opts...)
	if err != nil {
		return nil, nil, err
	}
	return userpb.NewUserClient(conn), conn, nil
}
