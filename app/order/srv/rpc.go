package srv

import (
	"context"
	"fmt"
	gpb "goshop/api/order/v1"
	"goshop/app/order/srv/config"
	"goshop/app/order/srv/internal/boundary"
	"goshop/app/order/srv/internal/controller/order/v1"
	db2 "goshop/app/order/srv/internal/data/v1/db"
	v13 "goshop/app/order/srv/internal/service/v1"
	"goshop/gmicro/core/trace"
	"goshop/gmicro/server/rpcserver"
)

func initOrderTrace(cfg *config.Config) error {
	return trace.InitAgent(trace.Options{
		Name:     cfg.Telemetry.Name,
		Endpoint: cfg.Telemetry.Endpoint,
		Sampler:  cfg.Telemetry.Sampler,
		Batcher:  cfg.Telemetry.Batcher,
	})
}

func newOrderServiceFactory(ctx context.Context, cfg *config.Config) (v13.ServiceFactory, error) {
	dataFactory, err := db2.GetDataFactoryOr(cfg.MySQLOptions)
	if err != nil {
		return nil, err
	}

	goodsGateway, err := boundary.NewGoodsRPCGatewayContext(ctx, cfg.Registry, cfg.RPC, cfg.RPCClientResilience)
	if err != nil {
		return nil, err
	}
	inventoryGateway, err := boundary.NewInventoryRPCGatewayContext(ctx, cfg.Registry, cfg.RPC, cfg.RPCClientResilience)
	if err != nil {
		return nil, err
	}

	return v13.NewService(dataFactory, cfg.Dtm, goodsGateway, inventoryGateway, cfg.Lifecycle.ToServiceConfig()), nil
}

func newOrderRPCServerWithFactory(cfg *config.Config, orderSrvFactory v13.ServiceFactory) (*rpcserver.Server, error) {
	orderServer := order.NewOrderServer(orderSrvFactory)
	rpcAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	tlsConfig, err := cfg.RPC.LoadServerTLSConfig()
	if err != nil {
		return nil, err
	}
	grpcServer, err := rpcserver.NewServerE(
		rpcserver.WithAddress(rpcAddr),
		rpcserver.WithMetrics(cfg.Server != nil && cfg.Server.EnableMetrics),
		rpcserver.WithServerTLSConfig(tlsConfig),
	)
	if err != nil {
		return nil, err
	}
	gpb.RegisterOrderServer(grpcServer.Server, orderServer)
	return grpcServer, nil
}

func NewOrderRPCServer(ctx context.Context, cfg *config.Config) (*rpcserver.Server, error) {
	if err := initOrderTrace(cfg); err != nil {
		return nil, err
	}

	orderSrvFactory, err := newOrderServiceFactory(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return newOrderRPCServerWithFactory(cfg, orderSrvFactory)
}
