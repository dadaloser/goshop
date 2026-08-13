package serverinterceptors

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testServerStream struct{ ctx context.Context }

func (s testServerStream) SetHeader(metadata.MD) error  { return nil }
func (s testServerStream) SendHeader(metadata.MD) error { return nil }
func (s testServerStream) SetTrailer(metadata.MD)       {}
func (s testServerStream) Context() context.Context     { return s.ctx }
func (s testServerStream) SendMsg(any) error            { return nil }
func (s testServerStream) RecvMsg(any) error            { return nil }

func TestStreamTimeoutInterceptorReturnsDeadlineExceeded(t *testing.T) {
	release := make(chan struct{})
	interceptor := StreamTimeoutInterceptor(10 * time.Millisecond)

	err := interceptor(nil, testServerStream{ctx: context.Background()},
		&grpc.StreamServerInfo{FullMethod: "/test.Service/Slow"},
		func(interface{}, grpc.ServerStream) error {
			<-release
			return nil
		})
	close(release)

	if got := status.Code(err); got != codes.DeadlineExceeded {
		t.Fatalf("status.Code(err) = %v, want DeadlineExceeded", got)
	}
}

func TestStreamTimeoutInterceptorPropagatesDeadlineContext(t *testing.T) {
	interceptor := StreamTimeoutInterceptor(10 * time.Millisecond)
	err := interceptor(nil, testServerStream{ctx: context.Background()},
		&grpc.StreamServerInfo{FullMethod: "/test.Service/Context"},
		func(_ interface{}, stream grpc.ServerStream) error {
			<-stream.Context().Done()
			return stream.Context().Err()
		})

	if got := status.Code(err); got != codes.DeadlineExceeded {
		t.Fatalf("status.Code(err) = %v, want DeadlineExceeded", got)
	}
}

func TestStreamConcurrencyInterceptorRejectsExcessStream(t *testing.T) {
	interceptor := StreamConcurrencyInterceptor(1)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- interceptor(nil, testServerStream{ctx: context.Background()}, nil,
			func(interface{}, grpc.ServerStream) error {
				close(entered)
				<-release
				return nil
			})
	}()
	<-entered

	err := interceptor(nil, testServerStream{ctx: context.Background()}, nil,
		func(interface{}, grpc.ServerStream) error { return nil })
	if got := status.Code(err); got != codes.ResourceExhausted {
		t.Fatalf("status.Code(err) = %v, want ResourceExhausted", got)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("admitted stream error = %v, want nil", err)
	}
}
