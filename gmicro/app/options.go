package app

import (
	"goshop/gmicro/registry"
	gs "goshop/gmicro/server"
	"goshop/gmicro/server/restserver"
	"goshop/gmicro/server/rpcserver"
	"net/url"
	"os"
	"time"
)

type Option func(o *options)

type options struct {
	id                        string
	endpoints                 []*url.URL
	healthCheckEndpoint       *url.URL
	healthCheckEndpointServer *restserver.Server
	name                      string
	version                   string

	sigs []os.Signal

	//允许用户传入自己的实现
	registrar        registry.Registrar
	registrarTimeout time.Duration

	//stop超时时间
	stopTimeout time.Duration

	restServer *restserver.Server
	rpcServer  *rpcserver.Server
	servers    []gs.Server
}

func WithRegistrar(registrar registry.Registrar) Option {
	return func(o *options) {
		o.registrar = registrar
	}
}

func WithEndpoints(endpoints []*url.URL) Option {
	return func(o *options) {
		o.endpoints = endpoints
	}
}

// WithHealthCheckEndpoint sets the endpoint used by service registries for
// active health checks. It is not advertised as a business service endpoint.
func WithHealthCheckEndpoint(endpoint *url.URL) Option {
	return func(o *options) {
		o.healthCheckEndpoint = endpoint
	}
}

// WithHealthCheckServer uses a REST server's listening endpoint for active
// registry health checks. The endpoint is read after the server reports ready.
func WithHealthCheckServer(server *restserver.Server) Option {
	return func(o *options) {
		o.healthCheckEndpointServer = server
	}
}

func WithRPCServer(server *rpcserver.Server) Option {
	return func(o *options) {
		o.rpcServer = server
	}
}

func WithServer(server gs.Server) Option {
	return func(o *options) {
		if server != nil {
			o.servers = append(o.servers, server)
		}
	}
}

func WithRestServer(server *restserver.Server) Option {
	return func(o *options) {
		o.restServer = server
	}
}

func WithID(id string) Option {
	return func(o *options) {
		o.id = id
	}
}

func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithVersion sets the service version registered with service discovery.
func WithVersion(version string) Option {
	return func(o *options) {
		if version != "" {
			o.version = version
		}
	}
}

func WithSigs(sigs []os.Signal) Option {
	return func(o *options) {
		o.sigs = sigs
	}
}

func WithRegistrarTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout > 0 {
			o.registrarTimeout = timeout
		}
	}
}

func WithStopTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout > 0 {
			o.stopTimeout = timeout
		}
	}
}
