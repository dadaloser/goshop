package srv

import (
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

func NewOrderRPCServer(cfg *config.Config) (*rpcserver.Server, error) {
	//初始化open-telemetry的exporter
	if err := trace.InitAgent(trace.Options{
		Name:     cfg.Telemetry.Name,
		Endpoint: cfg.Telemetry.Endpoint,
		Sampler:  cfg.Telemetry.Sampler,
		Batcher:  cfg.Telemetry.Batcher,
	}); err != nil {
		return nil, err
	}

	dataFactory, err := db2.GetDataFactoryOr(cfg.MySQLOptions)
	if err != nil {
		return nil, err
	}

	goodsGateway, err := boundary.NewGoodsRPCGateway(cfg.Registry)
	if err != nil {
		return nil, err
	}

	orderSrvFactory := v13.NewService(dataFactory, cfg.Dtm, goodsGateway)
	orderServer := order.NewOrderServer(orderSrvFactory)
	rpcAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	grpcServer, err := rpcserver.NewServerE(rpcserver.WithAddress(rpcAddr))
	if err != nil {
		return nil, err
	}
	gpb.RegisterOrderServer(grpcServer.Server, orderServer)
	return grpcServer, nil
}
