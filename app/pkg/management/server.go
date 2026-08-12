// Package management builds the internal HTTP server used by gRPC services
// for health checks, metrics, and optional profiling.
package management

import (
	"fmt"

	"goshop/app/pkg/options"
	"goshop/gmicro/server/restserver"
)

// NewServer creates a lifecycle-managed observability server for a gRPC
// service. The server listens on ServerOptions.HttpPort and only exposes
// built-in routes.
func NewServer(opts *options.ServerOptions, readinessChecks ...func() error) (*restserver.Server, error) {
	if opts == nil {
		return nil, fmt.Errorf("management server options are required")
	}
	if opts.HttpPort <= 0 {
		return nil, fmt.Errorf("server.http-port must be positive for the management server")
	}

	serverOpts := []restserver.ServerOption{
		restserver.WithPort(opts.HttpPort),
		restserver.WithHost(opts.Host),
		restserver.WithServiceName(opts.Name + "-management"),
		restserver.WithHealthCheck(opts.EnableHealthCheck),
		restserver.WithEnableProfiling(opts.EnableProfiling),
		restserver.WithProfilingToken(opts.ProfilingToken),
		restserver.WithMetrics(opts.EnableMetrics),
		restserver.WithBuiltInRouteCIDRs(opts.BuiltInRouteCIDRs),
		restserver.WithReadHeaderTimeout(opts.ReadHeaderTimeout),
		restserver.WithReadTimeout(opts.ReadTimeout),
		restserver.WithWriteTimeout(opts.WriteTimeout),
		restserver.WithIdleTimeout(opts.IdleTimeout),
	}
	for _, check := range readinessChecks {
		if check != nil {
			serverOpts = append(serverOpts, restserver.WithReadinessCheck(check))
		}
	}

	return restserver.NewServer(serverOpts...), nil
}
