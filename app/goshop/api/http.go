package admin

import (
	"goshop/app/goshop/api/config"
	"goshop/gmicro/server/restserver"
	"goshop/gmicro/server/restserver/middlewares"
)

func NewAPIHTTPServer(cfg *config.Config) (*restserver.Server, error) {
	opts := []restserver.ServerOption{
		restserver.WithPort(cfg.Server.HttpPort),
		restserver.WithHost(cfg.Server.Host),
		restserver.WithMiddlewares(cfg.Server.Middlewares),
		restserver.WithHealthCheck(cfg.Server.EnableHealthCheck),
		restserver.WithEnableProfiling(cfg.Server.EnableProfiling),
		restserver.WithProfilingToken(cfg.Server.ProfilingToken),
		restserver.WithMetrics(cfg.Server.EnableMetrics),
		restserver.WithReadHeaderTimeout(cfg.Server.ReadHeaderTimeout),
		restserver.WithReadTimeout(cfg.Server.ReadTimeout),
		restserver.WithWriteTimeout(cfg.Server.WriteTimeout),
		restserver.WithIdleTimeout(cfg.Server.IdleTimeout),
		restserver.WithCorsOptions(middlewares.CorsOptions{
			AllowOrigins: cfg.Server.CorsAllowOrigins,
		}),
		restserver.WithJwt(&restserver.JwtInfo{
			Realm:      cfg.Jwt.Realm,
			Key:        cfg.Jwt.Key,
			Timeout:    cfg.Jwt.Timeout,
			MaxRefresh: cfg.Jwt.MaxRefresh,
		}),
	}
	if cfg.Server.EnableLimit {
		opts = append(opts,
			restserver.WithRateLimit(cfg.Server.RateLimitRPS, cfg.Server.RateLimitBurst),
			restserver.WithMaxConcurrentRequests(cfg.Server.MaxConcurrentRequests),
		)
	}
	aRestServer := restserver.NewServer(opts...)

	//配置好路由
	initRouter(aRestServer, cfg)

	return aRestServer, nil
}
