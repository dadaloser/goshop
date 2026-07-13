package rpcserver

import (
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func WithServerTLSConfig(cfg *tls.Config) ServerOption {
	return func(s *Server) {
		if cfg == nil {
			return
		}
		s.tlsEnabled = true
		s.grpcOpts = append(s.grpcOpts, grpc.Creds(credentials.NewTLS(cfg)))
	}
}

func WithClientTLSConfig(cfg *tls.Config) ClientOption {
	return func(o *clientOptions) {
		if cfg == nil {
			return
		}
		o.transportCredentials = credentials.NewTLS(cfg)
	}
}
