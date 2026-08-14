package serverinterceptors

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryTimeoutInterceptorDoesNotDetachHandler(t *testing.T) {
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
		t.Fatalf("interceptor returned before handler exited: %v", err)
	case <-time.After(30 * time.Millisecond):
	}

	close(release)
	err := <-result
	if got := status.Code(err); got != codes.DeadlineExceeded {
		t.Fatalf("status.Code(err) = %v, want %v (err=%v)", got, codes.DeadlineExceeded, err)
	}
}

func TestUnaryTimeoutInterceptorCancelsContextForHandlerCleanup(t *testing.T) {
	interceptor := UnaryTimeoutInterceptor(10 * time.Millisecond)

	_, err := interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Cleanup"},
		func(ctx context.Context, _ interface{}) (interface{}, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	)
	if got := status.Code(err); got != codes.DeadlineExceeded {
		t.Fatalf("status.Code(err) = %v, want %v (err=%v)", got, codes.DeadlineExceeded, err)
	}
}
