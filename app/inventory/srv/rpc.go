package srv

import (
	"fmt"
	gpb "goshop/api/inventory/v1"
	"goshop/app/inventory/srv/config"
	v12 "goshop/app/inventory/srv/internal/controller/v1"
	db2 "goshop/app/inventory/srv/internal/data/v1/db"
	v13 "goshop/app/inventory/srv/internal/service/v1"
	"goshop/gmicro/core/trace"
	"goshop/gmicro/server/rpcserver"
)

func NewInventoryRPCServer(cfg *config.Config) (*rpcserver.Server, error) {
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
	invService := v13.NewService(dataFactory, cfg.RedisOptions)
	invServer := v12.NewInventoryServer(invService)
	rpcAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	grpcServer, err := rpcserver.NewServerE(rpcserver.WithAddress(rpcAddr))
	if err != nil {
		return nil, err
	}
	gpb.RegisterInventoryServer(grpcServer.Server, invServer)
	//r := gin.Default()
	//upb.RegisterUserServerHTTPServer(userver, r)
	//r.Run(":8075")
	return grpcServer, nil
}
