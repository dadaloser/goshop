package rpc

import (
	"context"
	"fmt"
	gpb "goshop/api/goods/v1"
	ipb "goshop/api/inventory/v1"
	opb "goshop/api/order/v1"
	upb "goshop/api/user/v1"
	"goshop/app/goshop/api/internal/data"
	appclient "goshop/app/pkg/client"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/rpcserver"
	errors2 "goshop/pkg/errors"
	"sync"
)

type grpcData struct {
	gc gpb.GoodsClient
	ic ipb.InventoryClient
	oc opb.OrderClient
	uc upb.UserClient
}

func (g grpcData) Goods() gpb.GoodsClient {
	return g.gc
}

func (g grpcData) Orders() opb.OrderClient {
	return g.oc
}

func (g grpcData) Inventory() ipb.InventoryClient {
	return g.ic
}

func (g grpcData) Users() data.UserData {
	return NewUsers(g.uc)
}

var (
	dbFactory data.DataFactory
	factoryMu sync.Mutex
)

// rpc的连接， 基于服务发现
func GetDataFactoryOr(ctx context.Context, options *options.RegistryOptions, rpcSecurity *options.RPCSecurityOptions) (data.DataFactory, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	if dbFactory != nil {
		return dbFactory, nil
	}
	if options == nil {
		return nil, fmt.Errorf("failed to get grpc store factory")
	}

	dialOpts := []rpcserver.ClientOption{
		rpcserver.WithConnectProbe(false),
	}

	userClient, _, err := appclient.NewUserClient(ctx, options, rpcSecurity, dialOpts...)
	if err != nil {
		return nil, errors2.WrapC(err, code.ErrConnectGRPC, "failed to get grpc store factory")
	}
	goodsClient, _, err := appclient.NewGoodsClient(ctx, options, rpcSecurity, dialOpts...)
	if err != nil {
		return nil, errors2.WrapC(err, code.ErrConnectGRPC, "failed to get grpc store factory")
	}
	inventoryClient, _, err := appclient.NewInventoryClient(ctx, options, rpcSecurity, dialOpts...)
	if err != nil {
		return nil, errors2.WrapC(err, code.ErrConnectGRPC, "failed to get grpc store factory")
	}
	orderClient, _, err := appclient.NewOrderClient(ctx, options, rpcSecurity, dialOpts...)
	if err != nil {
		return nil, errors2.WrapC(err, code.ErrConnectGRPC, "failed to get grpc store factory")
	}

	factory := &grpcData{
		gc: goodsClient,
		ic: inventoryClient,
		oc: orderClient,
		uc: userClient,
	}

	factoryMu.Lock()
	defer factoryMu.Unlock()
	if dbFactory == nil {
		dbFactory = factory
	}
	return dbFactory, nil
}
