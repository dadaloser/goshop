package srv

import (
	"goshop/app/order/srv/config"
	"goshop/app/pkg/options"
	gapp "goshop/gmicro/app"
	"goshop/gmicro/registry"
	"goshop/gmicro/registry/consul"
	"goshop/pkg/app"
	"goshop/pkg/log"

	"github.com/hashicorp/consul/api"
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
	r := consul.New(cli, consul.WithHealthCheck(true))
	return r, nil
}

func NeworderApp(cfg *config.Config) (*gapp.App, error) {
	//服务注册
	register, err := NewRegistrar(cfg.Registry)
	if err != nil {
		return nil, err
	}

	//生成rpc服务
	rpcServer, err := NewOrderRPCServer(cfg)
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
	return func(baseName string) error {
		log.Init(cfg.Log)
		defer log.Flush()

		orderApp, err := NeworderApp(cfg)
		if err != nil {
			return err
		}

		//启动
		if err := orderApp.Run(); err != nil {
			log.Errorf("run order app error: %s", err)
			return err
		}
		return nil
	}
}
