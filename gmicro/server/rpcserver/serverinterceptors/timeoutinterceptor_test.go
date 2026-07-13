package serverinterceptors

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryTimeoutInterceptorReturnsDeadlineExceededWithoutWaiting(t *testing.T) {
	release := make(chan struct{})
	result := make(chan error, 1)

	interceptor := UnaryTimeoutInterceptor(10 * time.Millisecond)
	go func() {
		_, err := interceptor(
			context.Background(),
			nil,
			&grpc.UnaryServerInfo{FullMethod: "/test.Service/Slow"},
			func(context.Context, interface{}) (interface{}, error) {
				<-release
				return "ok", nil
			},
		)
		result <- err
	}()

	select {
	case err := <-result:
		if got := status.Code(err); got != codes.DeadlineExceeded {
			t.Fatalf("status.Code(err) = %v, want %v (err=%v)", got, codes.DeadlineExceeded, err)
		}
	case <-time.After(time.Second):
		t.Fatal("UnaryTimeoutInterceptor did not return on timeout")
	}

	close(release)
}

func TestUnaryTimeoutInterceptorCancelsContextForHandlerCleanup(t *testing.T) {
	handlerDone := make(chan struct{})
	interceptor := UnaryTimeoutInterceptor(10 * time.Millisecond)

	_, err := interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Cleanup"},
		func(ctx context.Context, _ interface{}) (interface{}, error) {
			defer close(handlerDone)
			<-ctx.Done()
			return nil, ctx.Err()
		},
	)
	if got := status.Code(err); got != codes.DeadlineExceeded {
		t.Fatalf("status.Code(err) = %v, want %v (err=%v)", got, codes.DeadlineExceeded, err)
	}

	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("handler did not observe timeout context cancellation")
	}
}

func TestUnaryTimeoutInterceptorRecoversHandlerPanic(t *testing.T) {
	interceptor := UnaryTimeoutInterceptor(time.Second)

	_, err := interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Panic"},
		func(context.Context, interface{}) (interface{}, error) {
			panic("boom")
		},
	)
	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("status.Code(err) = %v, want %v (err=%v)", got, codes.Internal, err)
	}
}
