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

func TestUnaryConcurrencyInterceptorRejectsExcessRequestAndReleasesSlot(t *testing.T) {
	interceptor := UnaryConcurrencyInterceptor(1)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := interceptor(context.Background(), nil, nil, func(context.Context, interface{}) (interface{}, error) {
			close(entered)
			<-release
			return nil, nil
		})
		done <- err
	}()
	<-entered

	if _, err := interceptor(context.Background(), nil, nil, func(context.Context, interface{}) (interface{}, error) { return nil, nil }); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("saturated unary code = %v, want ResourceExhausted", status.Code(err))
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("admitted unary error = %v, want nil", err)
	}
	if _, err := interceptor(context.Background(), nil, nil, func(context.Context, interface{}) (interface{}, error) { return nil, nil }); err != nil {
		t.Fatalf("unary after release error = %v, want nil", err)
	}
}
