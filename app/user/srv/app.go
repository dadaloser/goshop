package srv

import (
	"context"

	"goshop/app/pkg/errorcatalog"
	"goshop/app/pkg/management"
	"goshop/app/pkg/options"
	"goshop/app/user/srv/config"
	"goshop/app/user/srv/internal/data/v1/db"
	userservice "goshop/app/user/srv/internal/service/v1"
	gapp "goshop/gmicro/app"
	"goshop/gmicro/registry"
	"goshop/gmicro/registry/consul"
	"goshop/gmicro/server/rpcserver"
	"goshop/pkg/app"
	"goshop/pkg/log"

	"github.com/google/wire"
	"github.com/hashicorp/consul/api"
	"golang.org/x/sync/errgroup"
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
	r := consul.New(cli, consul.WithHealthCheck(true), consul.WithHeartbeat(true), consul.WithHealthCheckInterval(1))
	return r, nil
}

func NewUserApp(
	register registry.Registrar,
	serverOpts *options.ServerOptions,
	registryOpts *options.RegistryOptions,
	rpcServer *rpcserver.Server,
) (*gapp.App, error) {
	log.Infof("creating user application: name=%s", serverOpts.Name)
	managementServer, err := management.NewServer(serverOpts)
	if err != nil {
		return nil, err
	}
	return gapp.New(
		gapp.WithName(serverOpts.Name),
		gapp.WithVersion(registryOpts.Version),
		gapp.WithRPCServer(rpcServer),
		gapp.WithServer(managementServer),
		gapp.WithHealthCheckServer(managementServer),
		gapp.WithRegistrar(register),
	), nil
}

func run(cfg *config.Config) app.RunFunc {
	return func(ctx context.Context, baseName string) error {
		errorcatalog.RegisterAll()
		log.Init(cfg.Log)
		defer log.Flush()

		log.Infof("initializing user service dependencies")
		userApp, err := initApp(cfg.Nacos, cfg.Server, cfg.Registry, cfg.RPC, cfg.Telemetry, cfg.MySQLOptions)
		if err != nil {
			return err
		}

		log.Infof("starting user service")
		group, groupCtx := errgroup.WithContext(ctx)
		group.Go(func() error { return userApp.RunContext(groupCtx) })
		if cfg.AccountDeletionEvents.Enabled() {
			gormDB, err := db.GetDBFactoryOr(cfg.MySQLOptions)
			if err != nil {
				return err
			}
			worker := userservice.NewAccountDeletionOutboxWorker(
				db.NewAccountDeletionOutboxStore(gormDB),
				userservice.AccountDeletionOutboxConfig{
					NATSURL:      cfg.AccountDeletionEvents.URL,
					PollInterval: cfg.AccountDeletionEvents.PollInterval,
					BatchSize:    cfg.AccountDeletionEvents.BatchSize,
					MaxRetries:   cfg.AccountDeletionEvents.MaxRetries,
				},
			)
			group.Go(func() error { return worker.Run(groupCtx) })
		}
		return group.Wait()
	}
}
