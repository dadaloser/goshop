package admin

import (
	"goshop/app/user/srv/config"
	"goshop/gmicro/server/restserver"
)

func NewUserHTTPServer(cfg *config.Config) (*restserver.Server, error) {
	restServer := restserver.NewServer(restserver.WithPort(cfg.Server.HttpPort),
		restserver.WithMiddlewares(cfg.Server.Middlewares),
		restserver.WithMetrics(true),
	)

	//配置好路由
	initRouter(restServer)

	return restServer, nil
}
