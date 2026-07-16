package srv

import (
	"context"

	"goshop/app/inventory/srv/config"
	"goshop/app/pkg/options"
	gapp "goshop/gmicro/app"
	"goshop/pkg/app"
	"goshop/pkg/log"
	"goshop/pkg/storage"

	"github.com/hashicorp/consul/api"

	"goshop/gmicro/registry"
	"goshop/gmicro/registry/consul"
)

func NewApp(basename string) *app.App {
	cfg := config.New()
	appl := app.NewApp("Inventory",
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

func NewInventoryApp(ctx context.Context, cfg *config.Config) (*gapp.App, error) {
	//服务注册
	register, err := NewRegistrar(cfg.Registry)
	if err != nil {
		return nil, err
	}

	//连接redis
	redisConfig := &storage.Config{
		Host:                  cfg.RedisOptions.Host,
		Port:                  cfg.RedisOptions.Port,
		Address:               cfg.RedisOptions.Addrs,
		MasterName:            cfg.RedisOptions.MasterName,
		Username:              cfg.RedisOptions.Username,
		Password:              cfg.RedisOptions.Password,
		Database:              cfg.RedisOptions.Database,
		MaxIdle:               cfg.RedisOptions.MaxIdle,
		MaxActive:             cfg.RedisOptions.MaxActive,
		Timeout:               cfg.RedisOptions.Timeout,
		EnableCluster:         cfg.RedisOptions.EnableCluster,
		UseSSL:                cfg.RedisOptions.UseSSL,
		SSLInsecureSkipVerify: cfg.RedisOptions.SSLInsecureSkipVerify,
		EnableTracing:         cfg.RedisOptions.EnableTracing,
		Resilience:            cfg.RedisOptions.Resilience,
	}
	go storage.ConnectToRedis(ctx, redisConfig)

	//生成rpc服务
	rpcServer, err := NewInventoryRPCServer(cfg)
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

		InventoryApp, err := NewInventoryApp(ctx, cfg)
		if err != nil {
			return err
		}

		//启动
		return InventoryApp.RunContext(ctx)
	}
}
