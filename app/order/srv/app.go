package srv

import (
	"context"

	"goshop/app/order/srv/config"
	v1 "goshop/app/order/srv/internal/service/v1"
	"goshop/app/pkg/options"
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
	appl := app.NewApp("order",
		"goshop",
		app.WithOptions(cfg),
		app.WithRunFunc(run(cfg)),
		//app.WithNoConfig(), //设置不读取配置文件
	)
	return appl
}

func NewRegistrar(registry *options.RegistryOptions) (registry.Registrar, error) {
	c := api.DefaultConfig()
	c.Address = registry.Address
	c.Scheme = registry.Scheme
	cli, err := api.NewClient(c)
	if err != nil {
		return nil, err
	}
	r := consul.New(cli, consul.WithHealthCheck(true), consul.WithHeartbeat(true), consul.WithHealthCheckInterval(1))
	return r, nil
}

func NeworderApp(ctx context.Context, cfg *config.Config) (*gapp.App, error) {
	if err := initOrderTrace(cfg); err != nil {
		return nil, err
	}
	orderSrvFactory, err := newOrderServiceFactory(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return newOrderAppWithServiceFactory(cfg, orderSrvFactory)
}

func newOrderAppWithServiceFactory(cfg *config.Config, orderSrvFactory v1.ServiceFactory) (*gapp.App, error) {
	//服务注册
	register, err := NewRegistrar(cfg.Registry)
	if err != nil {
		return nil, err
	}

	//生成rpc服务
	rpcServer, err := newOrderRPCServerWithFactory(cfg, orderSrvFactory)
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

		if err := initOrderTrace(cfg); err != nil {
			return err
		}

		orderSrvFactory, err := newOrderServiceFactory(ctx, cfg)
		if err != nil {
			return err
		}

		orderApp, err := newOrderAppWithServiceFactory(cfg, orderSrvFactory)
		if err != nil {
			return err
		}

		group, groupCtx := errgroup.WithContext(ctx)
		group.Go(func() error {
			return orderSrvFactory.RunBackground(groupCtx)
		})
		group.Go(func() error {
			return orderApp.RunContext(groupCtx)
		})
		return group.Wait()
	}
}
