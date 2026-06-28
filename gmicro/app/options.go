package app

import (
	"goshop/gmicro/registry"
	"goshop/gmicro/server/restserver"
	"goshop/gmicro/server/rpcserver"
	"net/url"
	"os"
	"time"
)

type Option func(o *options)

type options struct {
	id        string
	endpoints []*url.URL
	name      string

	sigs []os.Signal

	//允许用户传入自己的实现
	registrar        registry.Registrar
	registrarTimeout time.Duration

	//stop超时时间
	stopTimeout time.Duration

	restServer *restserver.Server
	rpcServer  *rpcserver.Server
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

func WithRPCServer(server *rpcserver.Server) Option {
	return func(o *options) {
		o.rpcServer = server
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
