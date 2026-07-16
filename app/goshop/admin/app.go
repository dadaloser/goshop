package admin

import (
	"context"

	"goshop/app/goshop/admin/config"
	"goshop/app/pkg/options"
	gapp "goshop/gmicro/app"
	"goshop/pkg/app"
	"goshop/pkg/log"

	"github.com/hashicorp/consul/api"

	"goshop/gmicro/registry"
	"goshop/gmicro/registry/consul"
)

func NewApp(basename string) *app.App {
	cfg := config.New()
	appl := app.NewApp("admin",
		basename,
		app.WithOptions(cfg),
		app.WithRunFunc(run(cfg)),
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
	r := consul.New(
		cli,
		consul.WithHealthCheck(true),
		consul.WithHeartbeat(false),
		consul.WithHTTPHealthCheckPath("/livez"),
	)
	return r, nil
}

func NewUserApp(cfg *config.Config) (*gapp.App, error) {
	//服务注册
	register, err := NewRegistrar(cfg.Registry)
	if err != nil {
		return nil, err
	}

	//生成rpc服务
	rpcServer, err := NewUserHTTPServer(cfg)
	if err != nil {
		return nil, err
	}

	managementServer := NewAdminManagementServer(cfg)

	opts := []gapp.Option{
		gapp.WithName(cfg.Server.Name),
		gapp.WithRestServer(rpcServer),
		gapp.WithRegistrar(register),
	}
	if managementServer != nil {
		opts = append(opts, gapp.WithServer(managementServer))
	}

	return gapp.New(opts...), nil
}

func run(cfg *config.Config) app.RunFunc {
	return func(ctx context.Context, baseName string) error {
		log.Init(cfg.Log)
		defer log.Flush()

		userApp, err := NewUserApp(cfg)
		if err != nil {
			return err
		}

		//启动
		return userApp.RunContext(ctx)
	}
}
