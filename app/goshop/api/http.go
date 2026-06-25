package admin

import (
	"goshop/app/goshop/api/config"
	"goshop/gmicro/server/restserver"
)

func NewAPIHTTPServer(cfg *config.Config) (*restserver.Server, error) {
	aRestServer := restserver.NewServer(restserver.WithPort(cfg.Server.HttpPort),
		restserver.WithMiddlewares(cfg.Server.Middlewares),
		restserver.WithMetrics(true),
	)

	//配置好路由
	initRouter(aRestServer, cfg)

	return aRestServer, nil
}
