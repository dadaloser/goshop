package admin

import (
	"context"
	"errors"
	"time"

	"goshop/app/goshop/admin/config"
	appclient "goshop/app/pkg/client"
	"goshop/gmicro/server/restserver"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/gmicro/server/rpcserver"
	"goshop/pkg/storage"
	"goshop/pkg/transport/httperror"
)

const adminStartupClientDialTimeout = 5 * time.Second

func NewUserHTTPServer(ctx context.Context, cfg *config.Config, clients ...*storage.Client) (*restserver.Server, error) {
	if ctx == nil {
		return nil, errors.New("admin HTTP server requires a startup context")
	}
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
	}
	if cfg.Server.EnableLimit {
		opts = append(opts,
			restserver.WithRateLimit(cfg.Server.RateLimitRPS, cfg.Server.RateLimitBurst),
			restserver.WithClientRouteRateLimit(cfg.Server.ClientRateLimitRPS, cfg.Server.ClientRateLimitBurst, cfg.Server.ClientRateLimitKeys),
			restserver.WithMaxConcurrentRequests(cfg.Server.MaxConcurrentRequests),
		)
	}
	restServer := restserver.NewServer(opts...)

	dialCtx, cancel := context.WithTimeout(ctx, adminStartupClientDialTimeout)
	userClient, _, err := appclient.NewUserClient(dialCtx, cfg.Registry, cfg.RPC)
	cancel()
	if err != nil {
		return nil, err
	}

	// 用户服务承载认证和授权，必须在启动时就绪；其余领域服务则允许
	// 延迟连接，以便后台在局部服务不可用时仍提供用户和 RBAC 功能。
	dialCtx, cancel = context.WithTimeout(ctx, adminStartupClientDialTimeout)
	goodsClient, _, err := appclient.NewGoodsClient(dialCtx, cfg.Registry, cfg.RPC, rpcserver.WithConnectProbe(false))
	cancel()
	if err != nil {
		return nil, err
	}
	dialCtx, cancel = context.WithTimeout(ctx, adminStartupClientDialTimeout)
	inventoryClient, _, err := appclient.NewInventoryClient(dialCtx, cfg.Registry, cfg.RPC, rpcserver.WithConnectProbe(false))
	cancel()
	if err != nil {
		return nil, err
	}
	dialCtx, cancel = context.WithTimeout(ctx, adminStartupClientDialTimeout)
	orderClient, _, err := appclient.NewOrderClient(dialCtx, cfg.Registry, cfg.RPC, rpcserver.WithConnectProbe(false))
	cancel()
	if err != nil {
		return nil, err
	}
	dialCtx, cancel = context.WithTimeout(ctx, adminStartupClientDialTimeout)
	reviewClient, _, err := appclient.NewReviewClient(dialCtx, cfg.Registry, cfg.RPC, rpcserver.WithConnectProbe(false))
	cancel()
	if err != nil {
		return nil, err
	}

	if err := initRouterWithBusinessClients(restServer, cfg, userClient, goodsClient, inventoryClient, orderClient, clients...); err != nil {
		return nil, err
	}
	if err := registerAdminReviewRoutes(restServer, cfg, userClient, goodsClient, reviewClient, clients...); err != nil {
		return nil, err
	}

	return restServer, nil
}

func NewAdminManagementServer(cfg *config.Config, clients ...*storage.Client) *restserver.Server {
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
