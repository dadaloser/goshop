package srv

import (
	"context"

	"goshop/app/goods/srv/config"
	db2 "goshop/app/goods/srv/internal/data/v1/db"
	"goshop/app/goods/srv/internal/data_search/v1/es"
	v1 "goshop/app/goods/srv/internal/service/v1"
	"goshop/app/pkg/options"
	gapp "goshop/gmicro/app"
	"goshop/pkg/app"
	"goshop/pkg/log"

	"github.com/hashicorp/consul/api"

	"golang.org/x/sync/errgroup"
	"goshop/gmicro/registry"
	"goshop/gmicro/registry/consul"
)

func NewApp(basename string) *app.App {
	cfg := config.New()
	appl := app.NewApp("goods",
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
	r := consul.New(cli, consul.WithHealthCheck(true), consul.WithHeartbeat(false))
	return r, nil
}

func NewGoodsApp(cfg *config.Config) (*gapp.App, error) {
	//服务注册
	register, err := NewRegistrar(cfg.Registry)
	if err != nil {
		return nil, err
	}

	//生成rpc服务
	rpcServer, err := NewGoodsRPCServer(cfg)
	if err != nil {
		return nil, err
	}

	return gapp.New(
		gapp.WithName(cfg.Server.Name),
		gapp.WithRPCServer(rpcServer),
		gapp.WithRegistrar(register),
	), nil
}

func newGoodsServiceFactory(cfg *config.Config) (v1.ServiceFactory, error) {
	dataFactory, err := db2.GetDBFactoryOr(cfg.MySQLOptions)
	if err != nil {
		return nil, err
	}

	searchFactory, err := es.GetSearchFactoryOr(cfg.EsOptions)
	if err != nil {
		return nil, err
	}
	return v1.NewService(dataFactory, searchFactory), nil
}

func run(cfg *config.Config) app.RunFunc {
	return func(ctx context.Context, baseName string) error {
		log.Init(cfg.Log)
		defer log.Flush()

		srvFactory, err := newGoodsServiceFactory(cfg)
		if err != nil {
			return err
		}

		goodsApp, err := NewGoodsApp(cfg)
		if err != nil {
			return err
		}

		group, groupCtx := errgroup.WithContext(ctx)
		group.Go(func() error {
			return srvFactory.RunBackground(groupCtx)
		})
		group.Go(func() error {
			return goodsApp.RunContext(groupCtx)
		})
		return group.Wait()
	}
}
