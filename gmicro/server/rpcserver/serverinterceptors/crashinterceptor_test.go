package serverinterceptors

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryCrashInterceptorReturnsInternalOnPanic(t *testing.T) {
	resp, err := UnaryCrashInterceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Panic"},
		func(context.Context, interface{}) (interface{}, error) {
			panic("boom")
		},
	)

	if resp != nil {
		t.Fatalf("UnaryCrashInterceptor() resp = %v, want nil", resp)
	}
	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("UnaryCrashInterceptor() code = %v, want %v", got, codes.Internal)
	}
}
