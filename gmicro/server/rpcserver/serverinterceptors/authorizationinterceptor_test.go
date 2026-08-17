package serverinterceptors

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestUnaryClientIdentityInterceptorAuthorizesExactURISAN(t *testing.T) {
	ctx := peerContextWithCertificate(t, "spiffe://goshop/api", nil)
	called := false
	_, err := UnaryClientIdentityInterceptor([]string{"spiffe://goshop/api"})(ctx, nil, nil,
		func(context.Context, interface{}) (interface{}, error) { called = true; return nil, nil })
	if err != nil {
		t.Fatalf("UnaryClientIdentityInterceptor(allowed URI) error = %v, want nil", err)
	}
	if !called {
		t.Fatal("authorized unary handler called = false, want true")
	}
}

func TestUnaryClientIdentityInterceptorRejectsMissingAndUnauthorizedIdentity(t *testing.T) {
	interceptor := UnaryClientIdentityInterceptor([]string{"spiffe://goshop/api"})
	if _, err := interceptor(context.Background(), nil, nil, func(context.Context, interface{}) (interface{}, error) { return nil, nil }); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing certificate code = %v, want Unauthenticated", status.Code(err))
	}
	ctx := peerContextWithCertificate(t, "spiffe://goshop/order", nil)
	if _, err := interceptor(ctx, nil, nil, func(context.Context, interface{}) (interface{}, error) { return nil, nil }); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("unauthorized certificate code = %v, want PermissionDenied", status.Code(err))
	}
}

func TestStreamClientIdentityInterceptorAuthorizesExactDNSSAN(t *testing.T) {
	ctx := peerContextWithCertificate(t, "", []string{"goshop-api.internal"})
	called := false
	err := StreamClientIdentityInterceptor([]string{"goshop-api.internal"})(nil, testServerStream{ctx: ctx}, nil,
		func(interface{}, grpc.ServerStream) error { called = true; return nil })
	if err != nil {
		t.Fatalf("StreamClientIdentityInterceptor(allowed DNS) error = %v, want nil", err)
	}
	if !called {
		t.Fatal("authorized stream handler called = false, want true")
	}
}

func peerContextWithCertificate(t *testing.T, uriSAN string, dnsSANs []string) context.Context {
	t.Helper()
	cert := &x509.Certificate{DNSNames: dnsSANs}
	if uriSAN != "" {
		uri, err := url.Parse(uriSAN)
		if err != nil {
			t.Fatalf("url.Parse(%q) error = %v", uriSAN, err)
		}
		cert.URIs = []*url.URL{uri}
	}
	return peer.NewContext(context.Background(), &peer.Peer{AuthInfo: credentials.TLSInfo{
		State: tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}},
	}})
}
