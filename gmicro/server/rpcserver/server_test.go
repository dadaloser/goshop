package rpcserver

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
)

func TestNewServerEReturnsListenError(t *testing.T) {
	_, err := NewServerE(WithAddress("127.0.0.1:-1"))
	if err == nil {
		t.Fatal("NewServerE() error = nil, want listen error")
	}
}

func TestNewServerEAddsStreamInterceptors(t *testing.T) {
	streamInterceptor := func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		return handler(srv, stream)
	}

	srv, err := NewServerE(
		WithAddress("127.0.0.1:0"),
		WithStreamInterceptor(streamInterceptor),
	)
	if err != nil {
		t.Fatalf("NewServerE() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	if len(srv.streamInts) != 1 {
		t.Fatalf("stream interceptors = %d, want 1", len(srv.streamInts))
	}
	if len(srv.grpcOpts) == 0 {
		t.Fatal("grpc options are empty, want stream interceptor option included")
	}
}

func TestServerReadyClosesAfterStart(t *testing.T) {
	srv, err := NewServerE(WithAddress("127.0.0.1:0"))
	if err != nil {
		t.Fatalf("NewServerE() error = %v, want nil", err)
	}
	go func() {
		_ = srv.Start(context.Background())
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	select {
	case <-srv.Ready():
	case <-time.After(time.Second):
		t.Fatal("Ready() was not closed after Start")
	}
}

func TestNewServerEDisablesReflectionByDefault(t *testing.T) {
	srv, err := NewServerE(WithAddress("127.0.0.1:0"))
	if err != nil {
		t.Fatalf("NewServerE() error = %v, want nil", err)
	}
	go func() {
		_ = srv.Start(context.Background())
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	conn, err := DialInsecure(
		context.Background(),
		WithEndpoint(srv.Endpoint().Host),
		WithConnectProbe(true),
		WithConnectTimeout(time.Second),
	)
	if err != nil {
		t.Fatalf("DialInsecure() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	client := reflectionv1.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(context.Background())
	if err != nil {
		t.Fatalf("ServerReflectionInfo() error = %v, want nil", err)
	}
	err = stream.Send(&reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_ListServices{},
	})
	if err != nil {
		t.Fatalf("reflection Send() error = %v, want nil", err)
	}
	resp, err := stream.Recv()
	if err == nil {
		t.Fatalf("reflection Recv() resp = %v, want error when reflection disabled", resp)
	}
	if err == io.EOF {
		t.Fatal("reflection Recv() error = EOF, want unimplemented/unavailable error")
	}
}

func TestNewServerEEnablesReflectionWhenConfigured(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp failed: %v", err)
	}
	srv, err := NewServerE(WithLis(lis), WithReflection(true))
	if err != nil {
		t.Fatalf("NewServerE() error = %v, want nil", err)
	}
	go func() {
		_ = srv.Start(context.Background())
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

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

	client := reflectionv1.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(context.Background())
	if err != nil {
		t.Fatalf("ServerReflectionInfo() error = %v, want nil", err)
	}
	if err := stream.Send(&reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_ListServices{},
	}); err != nil {
		t.Fatalf("reflection Send() error = %v, want nil", err)
	}
	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("reflection Recv() error = %v, want nil", err)
	}
	if resp.GetListServicesResponse() == nil {
		t.Fatalf("reflection response = %T, want list services response", resp.GetMessageResponse())
	}
}

func TestNewServerEAddsProductionGRPCOptions(t *testing.T) {
	srv, err := NewServerE(
		WithAddress("127.0.0.1:0"),
		WithMaxConcurrentStreams(128),
		WithKeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 10 * time.Second,
		}),
		WithKeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		t.Fatalf("NewServerE() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	if len(srv.grpcOpts) < 5 {
		t.Fatalf("grpc options = %d, want production options included", len(srv.grpcOpts))
	}
}

func TestNewServerEMarksSecureEndpointWhenTLSEnabled(t *testing.T) {
	serverTLS, _ := newTestMutualTLSConfigs(t, "goshop.internal")

	srv, err := NewServerE(
		WithAddress("127.0.0.1:0"),
		WithServerTLSConfig(serverTLS),
	)
	if err != nil {
		t.Fatalf("NewServerE() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	if srv.Endpoint() == nil {
		t.Fatal("Endpoint() = nil, want secure endpoint")
	}
	if !strings.Contains(srv.Endpoint().String(), "isSecure=true") {
		t.Fatalf("Endpoint() = %s, want secure endpoint query", srv.Endpoint().String())
	}
}
