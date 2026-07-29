package admin

import (
	"context"
	"errors"

	"goshop/app/goshop/admin/config"
	"goshop/app/pkg/errorcatalog"
	"goshop/app/pkg/options"
	gapp "goshop/gmicro/app"
	"goshop/pkg/app"
	"goshop/pkg/log"

	"github.com/hashicorp/consul/api"

	"goshop/gmicro/registry"
	"goshop/gmicro/registry/consul"
	"goshop/pkg/storage"
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
		consul.WithHTTPHealthCheckPath("/readyz"),
	)
	return r, nil
}

func NewUserApp(ctx context.Context, cfg *config.Config) (*gapp.App, error) {
	if ctx == nil {
		return nil, errors.New("admin app requires a run context")
	}
	//服务注册
	register, err := NewRegistrar(cfg.Registry)
	if err != nil {
		return nil, err
	}

	redisConfig := &storage.Config{
		Host:                  cfg.Redis.Host,
		Port:                  cfg.Redis.Port,
		Address:               cfg.Redis.Addrs,
		MasterName:            cfg.Redis.MasterName,
		Username:              cfg.Redis.Username,
		Password:              cfg.Redis.Password,
		Database:              cfg.Redis.Database,
		MaxIdle:               cfg.Redis.MaxIdle,
		MaxActive:             cfg.Redis.MaxActive,
		Timeout:               cfg.Redis.Timeout,
		EnableCluster:         cfg.Redis.EnableCluster,
		UseSSL:                cfg.Redis.UseSSL,
		SSLInsecureSkipVerify: cfg.Redis.SSLInsecureSkipVerify,
		EnableTracing:         cfg.Redis.EnableTracing,
		Resilience:            cfg.Redis.Resilience,
	}
	// Redis connectivity is a process-scoped background dependency probe. It is
	// intentionally tied to the application lifetime instead of an individual
	// request, so we reuse the app run context rather than a detached
	// detached root context.
	go storage.ConnectToRedis(ctx, redisConfig)

	//生成rpc服务
	rpcServer, err := NewUserHTTPServer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	managementServer := NewAdminManagementServer(cfg)

	opts := []gapp.Option{
		gapp.WithName(cfg.Server.Name),
		gapp.WithVersion(cfg.Registry.Version),
		gapp.WithRestServer(rpcServer),
		gapp.WithRegistrar(register),
	}
	if managementServer != nil {
		opts = append(opts,
			gapp.WithServer(managementServer),
			gapp.WithHealthCheckServer(managementServer),
		)
	}

	return gapp.New(opts...), nil
}

func run(cfg *config.Config) app.RunFunc {
	return func(ctx context.Context, baseName string) error {
		errorcatalog.RegisterAll()
		log.Init(cfg.Log)
		defer log.Flush()

		userApp, err := NewUserApp(ctx, cfg)
		if err != nil {
			return err
		}

		//启动
		return userApp.RunContext(ctx)
	}
}
