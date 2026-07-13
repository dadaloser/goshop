package api

import (
	"context"

	"goshop/app/goshop/api/config"
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
	appl := app.NewApp("api",
		"goshop",
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
	r := consul.New(cli, consul.WithHealthCheck(true), consul.WithHeartbeat(false))
	return r, nil
}

func NewAPIApp(ctx context.Context, cfg *config.Config) (*gapp.App, error) {
	//服务注册
	register, err := NewRegistrar(cfg.Registry)
	if err != nil {
		return nil, err
	}

	//连接redis
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
	go storage.ConnectToRedis(ctx, redisConfig)

	//生成http服务
	rpcServer, err := NewAPIHTTPServer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return gapp.New(
		gapp.WithName(cfg.Server.Name),
		gapp.WithRestServer(rpcServer),
		gapp.WithRegistrar(register),
	), nil
}

func run(cfg *config.Config) app.RunFunc {
	return func(ctx context.Context, baseName string) error {
		log.Init(cfg.Log)
		defer log.Flush()

		apiApp, err := NewAPIApp(ctx, cfg)
		if err != nil {
			return err
		}

		//启动
		return apiApp.RunContext(ctx)
	}
}
