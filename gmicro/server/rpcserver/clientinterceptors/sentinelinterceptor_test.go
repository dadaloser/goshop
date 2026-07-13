package clientinterceptors

import (
	"context"
	"testing"
	"time"

	"goshop/gmicro/resilience"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSentinelInterceptorIsolationFallback(t *testing.T) {
	options := resilience.NewOptions()
	options.MaxConcurrency = 1
	options.Timeout = time.Second
	interceptor, err := SentinelInterceptor(options)
	if err != nil {
		t.Fatalf("SentinelInterceptor() error = %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	firstResult := make(chan error, 1)
	invoker := func(ctx context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	go func() {
		firstResult <- interceptor(t.Context(), "/test.Service/Get", nil, nil, nil, invoker)
	}()
	<-started

	err = interceptor(t.Context(), "/test.Service/Get", nil, nil, nil, func(
		context.Context,
		string,
		interface{},
		interface{},
		*grpc.ClientConn,
		...grpc.CallOption,
	) error {
		return nil
	})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("second call code = %v, want Unavailable", status.Code(err))
	}
	close(release)
	if err := <-firstResult; err != nil {
		t.Fatalf("first call error = %v", err)
	}
}

func TestIsGRPCDependencyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unavailable", err: status.Error(codes.Unavailable, "down"), want: true},
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "not found", err: status.Error(codes.NotFound, "missing"), want: false},
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "invalid"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGRPCDependencyError(tt.err); got != tt.want {
				t.Fatalf("isGRPCDependencyError() = %v, want %v", got, tt.want)
			}
		})
	}
}
