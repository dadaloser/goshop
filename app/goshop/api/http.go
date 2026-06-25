package admin

import (
	"goshop/app/goshop/api/config"
	"goshop/gmicro/server/restserver"
)

func NewAPIHTTPServer(cfg *config.Config) (*restserver.Server, error) {
	aRestServer := restserver.NewServer(restserver.WithPort(cfg.Server.HttpPort),
		restserver.WithHost(cfg.Server.Host),
		restserver.WithMiddlewares(cfg.Server.Middlewares),
		restserver.WithHealthCheck(cfg.Server.EnableHealthCheck),
		restserver.WithEnableProfiling(cfg.Server.EnableProfiling),
		restserver.WithMetrics(cfg.Server.EnableMetrics),
		restserver.WithReadHeaderTimeout(cfg.Server.ReadHeaderTimeout),
		restserver.WithReadTimeout(cfg.Server.ReadTimeout),
		restserver.WithWriteTimeout(cfg.Server.WriteTimeout),
		restserver.WithIdleTimeout(cfg.Server.IdleTimeout),
	)

	//配置好路由
	initRouter(aRestServer, cfg)

	return aRestServer, nil
}
