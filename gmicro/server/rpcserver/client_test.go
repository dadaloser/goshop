package rpcserver

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"goshop/gmicro/registry"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestDialInsecureWithConnectProbeFailsWhenEndpointUnavailable(t *testing.T) {
	ctx := context.Background()
	conn, err := DialInsecure(
		ctx,
		WithEndpoint("127.0.0.1:1"),
		WithConnectProbe(true),
		WithConnectTimeout(20*time.Millisecond),
	)
	if err == nil {
		_ = conn.Close()
		t.Fatal("DialInsecure() error = nil, want connection probe error")
	}
	if conn != nil {
		t.Fatalf("DialInsecure() conn = %v, want nil on connection probe failure", conn)
	}
}

func TestDialInsecureWithConnectProbeSucceedsWhenEndpointReady(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp failed: %v", err)
	}

	server := grpc.NewServer()
	go func() {
		_ = server.Serve(lis)
	}()
	t.Cleanup(server.Stop)

	conn, err := DialInsecure(
		context.Background(),
		WithEndpoint(lis.Addr().String()),
		WithConnectProbe(true),
		WithConnectTimeout(time.Second),
	)
	if err != nil {
		t.Fatalf("DialInsecure() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
}

func TestDialDiscoveryInsecureRequiresDiscovery(t *testing.T) {
	conn, err := DialDiscoveryInsecure(
		context.Background(),
		WithEndpoint("discovery:///missing"),
		WithConnectProbe(false),
	)
	if err == nil {
		_ = conn.Close()
		t.Fatal("DialDiscoveryInsecure() error = nil, want missing discovery error")
	}
	if !strings.Contains(err.Error(), "discovery is required") {
		t.Fatalf("DialDiscoveryInsecure() error = %v, want discovery is required", err)
	}
}

func TestDialDiscoveryInsecureUsesDiscoveryDefaults(t *testing.T) {
	conn, err := DialDiscoveryInsecure(
		context.Background(),
		WithEndpoint("discovery:///missing"),
		WithDiscovery(fakeDiscovery{}),
		WithConnectProbe(false),
	)
	if err != nil {
		t.Fatalf("DialDiscoveryInsecure() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
}

func TestDialSecureRequiresTLSCredentials(t *testing.T) {
	conn, err := Dial(
		context.Background(),
		WithEndpoint("127.0.0.1:1"),
		WithConnectProbe(false),
	)
	if err == nil {
		_ = conn.Close()
		t.Fatal("Dial() error = nil, want missing TLS credentials error")
	}
	if !strings.Contains(err.Error(), "TLS credentials are required") {
		t.Fatalf("Dial() error = %v, want missing TLS credentials error", err)
	}
}

func TestDialSecureWithTLSConfigSucceedsWhenEndpointReady(t *testing.T) {
	serverTLS, clientTLS := newTestMutualTLSConfigs(t, "goshop.internal")

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp failed: %v", err)
	}

	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
	go func() {
		_ = server.Serve(lis)
	}()
	t.Cleanup(server.Stop)

	conn, err := Dial(
		context.Background(),
		WithEndpoint(lis.Addr().String()),
		WithClientTLSConfig(clientTLS),
		WithConnectProbe(true),
		WithConnectTimeout(time.Second),
	)
	if err != nil {
		t.Fatalf("Dial() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
}

func TestDialSecureWithSecurityPolicySucceedsWhenEndpointReady(t *testing.T) {
	policy := newTestSecurityPolicy(t, "goshop.internal")

	server, err := NewServerE(
		WithAddress("127.0.0.1:0"),
		WithServerSecurityPolicy(policy),
	)
	if err != nil {
		t.Fatalf("NewServerE() error = %v, want nil", err)
	}
	go func() {
		_ = server.Start(context.Background())
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})

	conn, err := Dial(
		context.Background(),
		WithEndpoint(server.Endpoint().Host),
		WithClientSecurityPolicy(policy),
		WithConnectProbe(true),
		WithConnectTimeout(time.Second),
	)
	if err != nil {
		t.Fatalf("Dial() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
}

type fakeDiscovery struct{}

func (fakeDiscovery) GetService(context.Context, string) ([]*registry.ServiceInstance, error) {
	return nil, nil
}

func (fakeDiscovery) Watch(context.Context, string) (registry.Watcher, error) {
	return fakeWatcher{}, nil
}

type fakeWatcher struct{}

func (fakeWatcher) Next() ([]*registry.ServiceInstance, error) {
	return nil, context.Canceled
}

func (fakeWatcher) Stop() error {
	return nil
}
