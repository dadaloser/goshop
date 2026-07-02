package rpc

import (
	"context"
	"fmt"
	gpb "goshop/api/goods/v1"
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
	uc upb.UserClient
}

func (g grpcData) Goods() gpb.GoodsClient {
	return g.gc
}

func (g grpcData) Users() data.UserData {
	return NewUsers(g.uc)
}

var (
	dbFactory data.DataFactory
	initErr   error
	once      sync.Once
)

// rpc的连接， 基于服务发现
func GetDataFactoryOr(options *options.RegistryOptions) (data.DataFactory, error) {
	if options == nil && dbFactory == nil {
		return nil, fmt.Errorf("failed to get grpc store factory")
	}

	//这里负责依赖的所有的rpc连接
	once.Do(func() {
		dialOpts := []rpcserver.ClientOption{
			rpcserver.WithConnectProbe(false),
		}

		userClient, _, err := appclient.NewUserClient(context.Background(), options, dialOpts...)
		if err != nil {
			initErr = err
			return
		}
		goodsClient, _, err := appclient.NewGoodsClient(context.Background(), options, dialOpts...)
		if err != nil {
			initErr = err
			return
		}

		dbFactory = &grpcData{
			gc: goodsClient,
			uc: userClient,
		}
	})

	if initErr != nil || dbFactory == nil {
		return nil, errors2.WrapC(initErr, code.ErrConnectGRPC, "failed to get grpc store factory")
	}
	return dbFactory, nil
}
