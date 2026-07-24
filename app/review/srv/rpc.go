package srv

import (
	"context"
	"fmt"

	rpb "goshop/api/review/v1"
	"goshop/app/pkg/client"
	"goshop/app/review/srv/config"
	db2 "goshop/app/review/srv/internal/data/db"
	"goshop/app/review/srv/internal/service"
	"goshop/gmicro/core/trace"
	"goshop/gmicro/server/rpcserver"
)

func initReviewTrace(cfg *config.Config) error {
	return trace.InitAgent(trace.Options{
		Name:     cfg.Telemetry.Name,
		Endpoint: cfg.Telemetry.Endpoint,
		Sampler:  cfg.Telemetry.Sampler,
		Batcher:  cfg.Telemetry.Batcher,
	})
}

func newReviewService(ctx context.Context, cfg *config.Config) (*Service, error) {
	if err := initReviewTrace(cfg); err != nil {
		return nil, err
	}

	db, err := db2.GetDBFactoryOr(cfg.MySQLOptions)
	if err != nil {
		return nil, err
	}
	orderClient, _, err := client.NewOrderClient(ctx, cfg.Registry, cfg.RPC, rpcserver.WithClientResilience(cfg.RPCClientResilience))
	if err != nil {
		return nil, err
	}
	workerCfg := cfg.Outbox.ToWorkerConfig()

	return New(
		NewStore(db),
		service.NewOrderVerifier(orderClient),
		WithOutboxWorker(workerCfg.PollInterval, workerCfg.BatchSize),
	), nil
}

func NewReviewRPCServer(cfg *config.Config, reviewService *Service) (*rpcserver.Server, error) {
	rpcAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	grpcServer, err := rpcserver.NewServerE(
		rpcserver.WithAddress(rpcAddr),
		rpcserver.WithMetrics(cfg.Server != nil && cfg.Server.EnableMetrics),
		rpcserver.WithServerSecurityPolicy(cfg.RPC),
	)
	if err != nil {
		return nil, err
	}
	rpb.RegisterReviewServer(grpcServer.Server, NewGRPCServer(reviewService))
	return grpcServer, nil
}
