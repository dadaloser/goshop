package rpcserver

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
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
