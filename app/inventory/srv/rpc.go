package srv

import (
	"fmt"
	gpb "goshop/api/inventory/v1"
	apimd "goshop/api/metadata"
	"goshop/app/inventory/srv/config"
	v12 "goshop/app/inventory/srv/internal/controller/v1"
	db2 "goshop/app/inventory/srv/internal/data/v1/db"
	v13 "goshop/app/inventory/srv/internal/service/v1"
	"goshop/app/pkg/grpcerror"
	"goshop/gmicro/core/trace"
	"goshop/gmicro/server/rpcserver"
	"goshop/pkg/storage"

	"google.golang.org/grpc"
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
	grpcServer, err := rpcserver.NewServerE(
		rpcserver.WithAddress(rpcAddr),
		rpcserver.WithRegistrar(func(server *grpc.Server) { apimd.RegisterMetadataServer(server, apimd.NewServer(server)) }),
		rpcserver.WithErrorMapper(grpcerror.Map),
		rpcserver.WithMetrics(cfg.Server != nil && cfg.Server.EnableMetrics),
		rpcserver.WithUnaryTimeout(cfg.Server.RPCUnaryTimeout),
		rpcserver.WithApplicationUnaryConcurrency(cfg.Server.RPCMaxConcurrentUnary),
		rpcserver.WithStreamMaxLifetime(cfg.Server.RPCStreamMaxLifetime),
		rpcserver.WithApplicationStreamConcurrency(cfg.Server.RPCMaxConcurrentStreams),
		rpcserver.WithServerSecurityPolicy(cfg.RPC),
		rpcserver.WithReadinessCheck(storage.ReadinessCheck()),
	)
	if err != nil {
		return nil, err
	}
	gpb.RegisterInventoryServer(grpcServer.Server, invServer)
	//r := gin.Default()
	//upb.RegisterUserServerHTTPServer(userver, r)
	//r.Run(":8075")
	return grpcServer, nil
}
