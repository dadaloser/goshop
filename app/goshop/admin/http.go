package admin

import (
	"goshop/app/goshop/admin/config"
	"goshop/gmicro/server/restserver"
	"goshop/gmicro/server/restserver/middlewares"
)

func NewUserHTTPServer(cfg *config.Config) (*restserver.Server, error) {
	enableBuiltInRoutes := cfg.Server.ManagementPort == 0
	opts := []restserver.ServerOption{
		restserver.WithPort(cfg.Server.HttpPort),
		restserver.WithHost(cfg.Server.Host),
		restserver.WithServiceName(cfg.Server.Name),
		restserver.WithMiddlewares(cfg.Server.Middlewares),
		restserver.WithHealthCheck(enableBuiltInRoutes && cfg.Server.EnableHealthCheck),
		restserver.WithEnableProfiling(enableBuiltInRoutes && cfg.Server.EnableProfiling),
		restserver.WithProfilingToken(cfg.Server.ProfilingToken),
		restserver.WithMetrics(enableBuiltInRoutes && cfg.Server.EnableMetrics),
		restserver.WithReadHeaderTimeout(cfg.Server.ReadHeaderTimeout),
		restserver.WithReadTimeout(cfg.Server.ReadTimeout),
		restserver.WithWriteTimeout(cfg.Server.WriteTimeout),
		restserver.WithIdleTimeout(cfg.Server.IdleTimeout),
		restserver.WithCorsOptions(middlewares.CorsOptions{
			AllowOrigins: cfg.Server.CorsAllowOrigins,
		}),
	}
	if cfg.Server.EnableLimit {
		opts = append(opts,
			restserver.WithRateLimit(cfg.Server.RateLimitRPS, cfg.Server.RateLimitBurst),
			restserver.WithMaxConcurrentRequests(cfg.Server.MaxConcurrentRequests),
		)
	}
	restServer := restserver.NewServer(opts...)

	//配置好路由
	initRouter(restServer, cfg)

	return restServer, nil
}

func NewAdminManagementServer(cfg *config.Config) *restserver.Server {
	if cfg == nil || cfg.Server == nil || cfg.Server.ManagementPort <= 0 {
		return nil
	}

	return restserver.NewServer(
		restserver.WithPort(cfg.Server.ManagementPort),
		restserver.WithHost(cfg.Server.Host),
		restserver.WithServiceName(cfg.Server.Name+"-management"),
		restserver.WithHealthCheck(cfg.Server.EnableHealthCheck),
		restserver.WithEnableProfiling(cfg.Server.EnableProfiling),
		restserver.WithProfilingToken(cfg.Server.ProfilingToken),
		restserver.WithMetrics(cfg.Server.EnableMetrics),
		restserver.WithBuiltInRouteCIDRs(cfg.Server.BuiltInRouteCIDRs),
		restserver.WithReadHeaderTimeout(cfg.Server.ReadHeaderTimeout),
		restserver.WithReadTimeout(cfg.Server.ReadTimeout),
		restserver.WithWriteTimeout(cfg.Server.WriteTimeout),
		restserver.WithIdleTimeout(cfg.Server.IdleTimeout),
	)
}
