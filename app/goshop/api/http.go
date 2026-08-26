package api

import (
	"context"

	"goshop/app/goshop/api/config"
	"goshop/gmicro/server/restserver"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/storage"
	"goshop/pkg/transport/httperror"
)

func NewAPIHTTPServer(ctx context.Context, cfg *config.Config, clients ...*storage.Client) (*restserver.Server, error) {
	readinessCheck := storage.UnavailableReadinessCheck()
	if len(clients) > 0 && clients[0] != nil {
		readinessCheck = clients[0].ReadinessCheck()
	}
	enableBuiltInRoutes := cfg.Server.ManagementPort == 0
	opts := []restserver.ServerOption{
		restserver.WithPort(cfg.Server.HttpPort),
		restserver.WithHost(cfg.Server.Host),
		restserver.WithServiceName(cfg.Server.Name),
		restserver.WithErrorResponder(httperror.AbortWithStatus),
		restserver.WithMiddlewares(cfg.Server.Middlewares),
		restserver.WithHealthCheck(enableBuiltInRoutes && cfg.Server.EnableHealthCheck),
		restserver.WithReadinessCheck(readinessCheck),
		restserver.WithEnableProfiling(enableBuiltInRoutes && cfg.Server.EnableProfiling),
		restserver.WithProfilingToken(cfg.Server.ProfilingToken),
		restserver.WithMetricsCollection(cfg.Server.EnableMetrics),
		restserver.WithMetricsEndpoint(enableBuiltInRoutes && cfg.Server.EnableMetrics),
		restserver.WithReadHeaderTimeout(cfg.Server.ReadHeaderTimeout),
		restserver.WithReadTimeout(cfg.Server.ReadTimeout),
		restserver.WithWriteTimeout(cfg.Server.WriteTimeout),
		restserver.WithIdleTimeout(cfg.Server.IdleTimeout),
		restserver.WithHandlerTimeout(cfg.Server.HTTPHandlerTimeout),
		restserver.WithMaxRequestBodyBytes(cfg.Server.MaxRequestBodyBytes),
		restserver.WithTrustedProxies(cfg.Server.TrustedProxies),
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
			restserver.WithClientRouteRateLimit(cfg.Server.ClientRateLimitRPS, cfg.Server.ClientRateLimitBurst, cfg.Server.ClientRateLimitKeys),
			restserver.WithMaxConcurrentRequests(cfg.Server.MaxConcurrentRequests),
		)
	}
	aRestServer := restserver.NewServer(opts...)

	//配置好路由
	if err := initRouter(ctx, aRestServer, cfg, clients...); err != nil {
		return nil, err
	}

	return aRestServer, nil
}

func NewAPIManagementServer(cfg *config.Config, clients ...*storage.Client) *restserver.Server {
	if cfg == nil || cfg.Server == nil || cfg.Server.ManagementPort <= 0 {
		return nil
	}

	readinessCheck := storage.UnavailableReadinessCheck()
	if len(clients) > 0 && clients[0] != nil {
		readinessCheck = clients[0].ReadinessCheck()
	}
	return restserver.NewServer(
		restserver.WithPort(cfg.Server.ManagementPort),
		restserver.WithHost(cfg.Server.Host),
		restserver.WithServiceName(cfg.Server.Name+"-management"),
		restserver.WithHealthCheck(cfg.Server.EnableHealthCheck),
		restserver.WithReadinessCheck(readinessCheck),
		restserver.WithEnableProfiling(cfg.Server.EnableProfiling),
		restserver.WithProfilingToken(cfg.Server.ProfilingToken),
		restserver.WithMetricsCollection(false),
		restserver.WithMetricsEndpoint(cfg.Server.EnableMetrics),
		restserver.WithBuiltInRouteCIDRs(cfg.Server.BuiltInRouteCIDRs),
		restserver.WithReadHeaderTimeout(cfg.Server.ReadHeaderTimeout),
		restserver.WithReadTimeout(cfg.Server.ReadTimeout),
		restserver.WithWriteTimeout(cfg.Server.WriteTimeout),
		restserver.WithIdleTimeout(cfg.Server.IdleTimeout),
		restserver.WithTrustedProxies(cfg.Server.TrustedProxies),
	)
}
