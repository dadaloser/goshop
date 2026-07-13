package client

import (
	"context"
	"fmt"
	goodspb "goshop/api/goods/v1"
	inventorypb "goshop/api/inventory/v1"
	orderpb "goshop/api/order/v1"
	userpb "goshop/api/user/v1"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/rpcserver"
	"goshop/gmicro/server/rpcserver/clientinterceptors"

	"google.golang.org/grpc"
)

func DialService(
	ctx context.Context,
	registry *options.RegistryOptions,
	security *options.RPCSecurityOptions,
	service string,
	opts ...rpcserver.ClientOption,
) (*grpc.ClientConn, error) {
	discovery, err := NewConsulDiscovery(registry)
	if err != nil {
		return nil, fmt.Errorf("create discovery for %s: %w", service, err)
	}
	if security == nil {
		return nil, fmt.Errorf("rpc security for %s is required", service)
	}
	tlsConfig, err := security.LoadClientTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("load rpc TLS config for %s: %w", service, err)
	}

	dialOpts := []rpcserver.ClientOption{
		rpcserver.WithEndpoint(ServiceEndpoint(service)),
		rpcserver.WithDiscovery(discovery),
		rpcserver.WithClientTLSConfig(tlsConfig),
		rpcserver.WithClientUnaryInterceptor(clientinterceptors.UnaryTracingInterceptor),
	}
	dialOpts = append(dialOpts, opts...)
	conn, err := rpcserver.DialDiscovery(ctx, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s (%s): %w", service, ServiceEndpoint(service), err)
	}
	return conn, nil
}

func NewGoodsClient(
	ctx context.Context,
	registry *options.RegistryOptions,
	security *options.RPCSecurityOptions,
	opts ...rpcserver.ClientOption,
) (goodspb.GoodsClient, *grpc.ClientConn, error) {
	conn, err := DialService(ctx, registry, security, ServiceGoods, opts...)
	if err != nil {
		return nil, nil, err
	}
	return goodspb.NewGoodsClient(conn), conn, nil
}

func NewInventoryClient(
	ctx context.Context,
	registry *options.RegistryOptions,
	security *options.RPCSecurityOptions,
	opts ...rpcserver.ClientOption,
) (inventorypb.InventoryClient, *grpc.ClientConn, error) {
	conn, err := DialService(ctx, registry, security, ServiceInventory, opts...)
	if err != nil {
		return nil, nil, err
	}
	return inventorypb.NewInventoryClient(conn), conn, nil
}

func NewOrderClient(
	ctx context.Context,
	registry *options.RegistryOptions,
	security *options.RPCSecurityOptions,
	opts ...rpcserver.ClientOption,
) (orderpb.OrderClient, *grpc.ClientConn, error) {
	conn, err := DialService(ctx, registry, security, ServiceOrder, opts...)
	if err != nil {
		return nil, nil, err
	}
	return orderpb.NewOrderClient(conn), conn, nil
}

func NewUserClient(
	ctx context.Context,
	registry *options.RegistryOptions,
	security *options.RPCSecurityOptions,
	opts ...rpcserver.ClientOption,
) (userpb.UserClient, *grpc.ClientConn, error) {
	conn, err := DialService(ctx, registry, security, ServiceUser, opts...)
	if err != nil {
		return nil, nil, err
	}
	return userpb.NewUserClient(conn), conn, nil
}
