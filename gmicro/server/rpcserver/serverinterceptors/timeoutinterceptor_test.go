package serverinterceptors

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestUnaryTimeoutInterceptorWaitsForHandlerToReturn(t *testing.T) {
	release := make(chan struct{})
	returned := make(chan struct{})

	interceptor := UnaryTimeoutInterceptor(10 * time.Millisecond)
	go func() {
		_, _ = interceptor(
			context.Background(),
			nil,
			&grpc.UnaryServerInfo{FullMethod: "/test.Service/Slow"},
			func(context.Context, interface{}) (interface{}, error) {
				<-release
				return "ok", nil
			},
		)
		close(returned)
	}()

	select {
	case <-returned:
		t.Fatal("UnaryTimeoutInterceptor returned before handler finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("UnaryTimeoutInterceptor did not return after handler finished")
	}
}
