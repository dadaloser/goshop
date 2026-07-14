package rpcserver

import (
	"crypto/tls"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	errServerSecurityAlreadyConfigured = errors.New("rpc server security is already configured")
	errClientSecurityAlreadyConfigured = errors.New("rpc client security is already configured")
)

func WithServerTLSConfig(cfg *tls.Config) ServerOption {
	return func(s *Server) {
		applyServerTLSConfig(s, cfg)
	}
}

func WithClientTLSConfig(cfg *tls.Config) ClientOption {
	return func(o *clientOptions) {
		applyClientTLSConfig(o, cfg)
	}
}

func WithServerSecurityPolicy(policy *SecurityPolicy) ServerOption {
	return func(s *Server) {
		if s == nil || policy == nil {
			return
		}
		s.securityPolicy = policy
	}
}

func WithClientSecurityPolicy(policy *SecurityPolicy) ClientOption {
	return func(o *clientOptions) {
		if o == nil || policy == nil {
			return
		}
		o.securityPolicy = policy
	}
}

func applyServerTLSConfig(s *Server, cfg *tls.Config) {
	if s == nil || cfg == nil {
		return
	}
	s.tlsEnabled = true
	s.grpcOpts = append(s.grpcOpts, grpc.Creds(credentials.NewTLS(cfg)))
}

func applyClientTLSConfig(o *clientOptions, cfg *tls.Config) {
	if o == nil || cfg == nil {
		return
	}
	o.transportCredentials = credentials.NewTLS(cfg)
}
