package srv

import (
	"context"

	"goshop/app/pkg/options"
	"goshop/app/review/srv/config"
	gapp "goshop/gmicro/app"
	"goshop/gmicro/registry"
	"goshop/gmicro/registry/consul"
	"goshop/pkg/app"
	"goshop/pkg/log"

	"github.com/hashicorp/consul/api"
	"golang.org/x/sync/errgroup"
)

func NewApp(basename string) *app.App {
	cfg := config.New()
	return app.NewApp("review",
		"goshop",
		app.WithOptions(cfg),
		app.WithRunFunc(run(cfg)),
	)
}

func NewRegistrar(registryOptions *options.RegistryOptions) (registry.Registrar, error) {
	c := api.DefaultConfig()
	c.Address = registryOptions.Address
	c.Scheme = registryOptions.Scheme
	cli, err := api.NewClient(c)
	if err != nil {
		return nil, err
	}
	return consul.New(cli, consul.WithHealthCheck(true), consul.WithHeartbeat(true), consul.WithHealthCheckInterval(1)), nil
}

func NewReviewApp(cfg *config.Config, reviewService *Service) (*gapp.App, error) {
	register, err := NewRegistrar(cfg.Registry)
	if err != nil {
		return nil, err
	}

	rpcServer, err := NewReviewRPCServer(cfg, reviewService)
	if err != nil {
		return nil, err
	}

	return gapp.New(
		gapp.WithName(cfg.Server.Name),
		gapp.WithRPCServer(rpcServer),
		gapp.WithRegistrar(register),
	), nil
}

func run(cfg *config.Config) app.RunFunc {
	return func(ctx context.Context, baseName string) error {
		log.Init(cfg.Log)
		defer log.Flush()

		reviewService, err := newReviewService(ctx, cfg)
		if err != nil {
			return err
		}

		reviewApp, err := NewReviewApp(cfg, reviewService)
		if err != nil {
			return err
		}

		group, groupCtx := errgroup.WithContext(ctx)
		group.Go(func() error {
			return reviewService.RunOutbox(groupCtx)
		})
		group.Go(func() error {
			return reviewApp.RunContext(groupCtx)
		})
		return group.Wait()
	}
}
