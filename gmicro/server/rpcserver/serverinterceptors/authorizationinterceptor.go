package serverinterceptors

import (
	"context"
	"crypto/x509"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func UnaryClientIdentityInterceptor(allowed []string) grpc.UnaryServerInterceptor {
	authorize := newClientIdentityAuthorizer(allowed)
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := authorize(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func StreamClientIdentityInterceptor(allowed []string) grpc.StreamServerInterceptor {
	authorize := newClientIdentityAuthorizer(allowed)
	return func(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := authorize(stream.Context()); err != nil {
			return err
		}
		return handler(srv, stream)
	}
}

func newClientIdentityAuthorizer(allowed []string) func(context.Context) error {
	allowset := make(map[string]struct{}, len(allowed))
	for _, identity := range allowed {
		if identity = strings.TrimSpace(identity); identity != "" {
			allowset[identity] = struct{}{}
		}
	}
	return func(ctx context.Context) error {
		p, ok := peer.FromContext(ctx)
		if !ok || p.AuthInfo == nil {
			return status.Error(codes.Unauthenticated, "verified client certificate required")
		}
		tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
		if !ok || len(tlsInfo.State.PeerCertificates) == 0 {
			return status.Error(codes.Unauthenticated, "verified client certificate required")
		}
		if certificateHasAllowedIdentity(tlsInfo.State.PeerCertificates[0], allowset) {
			return nil
		}
		return status.Error(codes.PermissionDenied, "client service identity is not authorized")
	}
}

func certificateHasAllowedIdentity(cert *x509.Certificate, allowed map[string]struct{}) bool {
	if cert == nil || len(allowed) == 0 {
		return false
	}
	for _, uri := range cert.URIs {
		if _, ok := allowed[uri.String()]; ok {
			return true
		}
	}
	for _, dnsName := range cert.DNSNames {
		if _, ok := allowed[dnsName]; ok {
			return true
		}
	}
	return false
}
