package serverinterceptors

import (
	"context"
	"testing"

	"goshop/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryErrorInterceptorConvertsProjectError(t *testing.T) {
	resp, err := UnaryErrorInterceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Error"},
		func(context.Context, interface{}) (interface{}, error) {
			return nil, errors.WithCode(1, "database exploded")
		},
	)

	if resp != nil {
		t.Fatalf("UnaryErrorInterceptor() resp = %v, want nil", resp)
	}
	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("UnaryErrorInterceptor() code = %v, want %v", got, codes.Internal)
	}
}

func TestUnaryErrorInterceptorPreservesStatusError(t *testing.T) {
	_, err := UnaryErrorInterceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/NotFound"},
		func(context.Context, interface{}) (interface{}, error) {
			return nil, status.Error(codes.NotFound, "missing")
		},
	)

	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("UnaryErrorInterceptor() code = %v, want %v", got, codes.NotFound)
	}
}
