package srv

import (
	"fmt"
	gpb "goshop/api/goods/v1"
	"goshop/app/goods/srv/config"
	v12 "goshop/app/goods/srv/internal/controller/v1"
	db2 "goshop/app/goods/srv/internal/data/v1/db"
	"goshop/app/goods/srv/internal/data_search/v1/es"
	v1 "goshop/app/goods/srv/internal/service/v1"
	"goshop/gmicro/core/trace"
	"goshop/gmicro/server/rpcserver"
)

func NewGoodsRPCServer(cfg *config.Config) (*rpcserver.Server, error) {
	//初始化open-telemetry的exporter
	if err := trace.InitAgent(trace.Options{
		Name:     cfg.Telemetry.Name,
		Endpoint: cfg.Telemetry.Endpoint,
		Sampler:  cfg.Telemetry.Sampler,
		Batcher:  cfg.Telemetry.Batcher,
	}); err != nil {
		return nil, err
	}

	//有点繁琐，wire， ioc-golang
	dataFactory, err := db2.GetDBFactoryOr(cfg.MySQLOptions)
	if err != nil {
		return nil, err
	}

	//构建，繁琐 - 工厂模式
	searchFactory, err := es.GetSearchFactoryOr(cfg.EsOptions)
	if err != nil {
		return nil, err
	}

	srvFactory := v1.NewService(dataFactory, searchFactory)
	goodsServer := v12.NewGoodsServer(srvFactory)
	rpcAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	tlsConfig, err := cfg.RPC.LoadServerTLSConfig()
	if err != nil {
		return nil, err
	}
	grpcServer, err := rpcserver.NewServerE(
		rpcserver.WithAddress(rpcAddr),
		rpcserver.WithServerTLSConfig(tlsConfig),
	)
	if err != nil {
		return nil, err
	}

	gpb.RegisterGoodsServer(grpcServer.Server, goodsServer)

	//r := gin.Default()
	//upb.RegisterUserServerHTTPServer(userver, r)
	//r.Run(":8075")
	return grpcServer, nil
}
