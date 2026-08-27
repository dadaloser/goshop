package rpcserver

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
)

func TestNewServerDefersListenErrorsUntilStart(t *testing.T) {
	srv, err := NewServer(WithAddress("127.0.0.1:-1"))
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	if endpoint := srv.Endpoint(); endpoint != nil {
		t.Fatalf("Endpoint() = %v before Start, want nil", endpoint)
	}
	if err := srv.Start(t.Context()); err == nil {
		t.Fatal("Start() error = nil, want listen error")
	}
}

func TestStopClosesInjectedListenerBeforeStart(t *testing.T) {
	lis := &trackedListener{}
	srv, err := NewServer(WithLis(lis))
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v, want nil", err)
	}
	if !lis.closed {
		t.Fatal("injected listener was not closed")
	}
}

func TestStopPreventsStart(t *testing.T) {
	srv, err := NewServer(WithAddress("127.0.0.1:0"))
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v, want nil", err)
	}
	if err := srv.Start(context.Background()); !errors.Is(err, errServerStopped) {
		t.Fatalf("Start() error = %v, want errServerStopped", err)
	}
}

type trackedListener struct {
	closed bool
}

func (l *trackedListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }

func (l *trackedListener) Close() error {
	l.closed = true
	return nil
}

func (*trackedListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
}

func TestNewServerAddsStreamInterceptors(t *testing.T) {
	streamInterceptor := func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		return handler(srv, stream)
	}

	srv, err := NewServer(
		WithAddress("127.0.0.1:0"),
		WithStreamInterceptor(streamInterceptor),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
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
	srv, err := NewServer(WithAddress("127.0.0.1:0"))
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
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

func TestNewServerDisablesReflectionByDefault(t *testing.T) {
	srv, err := NewServer(WithAddress("127.0.0.1:0"))
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
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
		// A missing bidirectional-stream handler can surface as EOF while the
		// client sends its first message, depending on the gRPC transport timing.
		// That is still the expected observable behavior when reflection is off.
		if err == io.EOF {
			return
		}
		t.Fatalf("reflection Send() error = %v, want EOF or a response error", err)
	}
	resp, err := stream.Recv()
	if err == nil {
		t.Fatalf("reflection Recv() resp = %v, want error when reflection disabled", resp)
	}
}

func TestNewServerEnablesReflectionWhenConfigured(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp failed: %v", err)
	}
	srv, err := NewServer(WithLis(lis), WithReflection(true))
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
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

func TestNewServerAddsProductionGRPCOptions(t *testing.T) {
	srv, err := NewServer(WithAddress("127.0.0.1:0"))
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
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

func TestWithProductionDefaultsDoesNotDuplicateGRPCOptions(t *testing.T) {
	defaultSrv, err := NewServer(WithAddress("127.0.0.1:0"))
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = defaultSrv.Stop(ctx)
	})

	explicitSrv, err := NewServer(
		WithAddress("127.0.0.1:0"),
		WithProductionDefaults(),
	)
	if err != nil {
		t.Fatalf("NewServer() with WithProductionDefaults error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = explicitSrv.Stop(ctx)
	})

	if got, want := len(explicitSrv.grpcOpts), len(defaultSrv.grpcOpts); got != want {
		t.Fatalf("grpc options = %d, want %d without duplicate production defaults", got, want)
	}
}

func TestNewServerEnablesMetricsByDefault(t *testing.T) {
	srv, err := NewServer(WithAddress("127.0.0.1:0"))
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	if !srv.enableMetrics {
		t.Fatal("enableMetrics = false, want true by default")
	}
}

func TestWithMetricsCanDisableMetrics(t *testing.T) {
	srv, err := NewServer(
		WithAddress("127.0.0.1:0"),
		WithMetrics(false),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	if srv.enableMetrics {
		t.Fatal("enableMetrics = true, want false when explicitly disabled")
	}
}

func TestUnaryAndStreamTimeoutsAreConfiguredIndependently(t *testing.T) {
	srv, err := NewServer(
		WithAddress("127.0.0.1:0"),
		WithUnaryTimeout(15*time.Second),
		WithStreamMaxLifetime(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	if got := srv.unaryTimeout; got != 15*time.Second {
		t.Fatalf("unaryTimeout = %s, want 15s", got)
	}
	if got := srv.streamMaxLifetime; got != 5*time.Minute {
		t.Fatalf("streamMaxLifetime = %s, want 5m", got)
	}
}

func TestWithTimeoutOnlyConfiguresUnaryRPCs(t *testing.T) {
	srv, err := NewServer(
		WithAddress("127.0.0.1:0"),
		WithTimeout(15*time.Second),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	if got := srv.unaryTimeout; got != 15*time.Second {
		t.Fatalf("unaryTimeout = %s, want 15s", got)
	}
	if got := srv.streamMaxLifetime; got != 0 {
		t.Fatalf("streamMaxLifetime = %s, want zero", got)
	}
}

func TestNewServerMarksTLSEnabled(t *testing.T) {
	serverTLS, _ := newTestMutualTLSConfigs(t, "service.example.test")

	srv, err := NewServer(
		WithAddress("127.0.0.1:0"),
		WithServerTLSConfig(serverTLS),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	if !srv.tlsEnabled {
		t.Fatal("tlsEnabled = false, want true")
	}
}

func TestNewServerWithServerSecurityPolicyMarksTLSEnabled(t *testing.T) {
	policy := newTestSecurityPolicy(t, "service.example.test")

	srv, err := NewServer(
		WithAddress("127.0.0.1:0"),
		WithServerSecurityPolicy(policy),
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	})

	if !srv.tlsEnabled {
		t.Fatal("tlsEnabled = false, want true")
	}
}
