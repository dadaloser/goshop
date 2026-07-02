package srv

import (
	"goshop/app/pkg/options"
	"goshop/app/user/srv/config"
	gapp "goshop/gmicro/app"
	"goshop/gmicro/server/rpcserver"
	"goshop/pkg/app"
	"goshop/pkg/log"

	"github.com/google/wire"
	"github.com/hashicorp/consul/api"

	"goshop/gmicro/registry"
	"goshop/gmicro/registry/consul"
)

var ProviderSet = wire.NewSet(NewUserApp, NewRegistrar, NewUserRPCServer)

func NewApp(basename string) *app.App {
	cfg := config.New()
	a := app.NewApp("user",
		"goshop",
		app.WithOptions(cfg),
		app.WithRunFunc(run(cfg)),
		//app.WithNoConfig(), //设置不读取配置文件,使用命令启动
	)
	return a
}

func NewRegistrar(registry *options.RegistryOptions) (registry.Registrar, error) {
	log.Infof("initializing consul registrar: address=%s scheme=%s", registry.Address, registry.Scheme)
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

func NewUserApp(register registry.Registrar,
	serverOpts *options.ServerOptions, rpcServer *rpcserver.Server) (*gapp.App, error) {
	log.Infof("creating user application: name=%s", serverOpts.Name)
	return gapp.New(
		gapp.WithName(serverOpts.Name),
		gapp.WithRPCServer(rpcServer),
		gapp.WithRegistrar(register),
	), nil
}

func run(cfg *config.Config) app.RunFunc {
	return func(baseName string) error {
		log.Init(cfg.Log)
		defer log.Flush()

		log.Infof("initializing user service dependencies")
		userApp, err := initApp(cfg.Nacos, cfg.Server, cfg.Registry, cfg.Telemetry, cfg.MySQLOptions)
		if err != nil {
			return err
		}

		//启动
		log.Infof("starting user service")
		if err := userApp.Run(); err != nil {
			log.Errorf("run user app error: %+v", err)
			return err
		}
		return nil
	}
}
