package serverinterceptors

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestUnaryErrorInterceptorUsesDefaultMapper(t *testing.T) {
	resp, err := UnaryErrorInterceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Error"},
		func(context.Context, interface{}) (interface{}, error) {
			return nil, errors.New("database exploded")
		},
	)

	if resp != nil {
		t.Fatalf("UnaryErrorInterceptor() resp = %v, want nil", resp)
	}
	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("UnaryErrorInterceptor() code = %v, want %v", got, codes.Internal)
	}
}

func TestUnaryErrorInterceptorWithMapper(t *testing.T) {
	mapper := func(err error) error {
		if err == nil {
			return nil
		}
		return status.Error(codes.NotFound, "mapped")
	}
	_, err := UnaryErrorInterceptorWithMapper(mapper)(context.Background(), nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		return nil, errors.New("missing")
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("UnaryErrorInterceptorWithMapper() code = %v, want %v", got, codes.NotFound)
	}
}

func TestDefaultErrorMapperPreservesTransportAndContextErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "canceled", err: context.Canceled, want: codes.Canceled},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: codes.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status.Code(DefaultErrorMapper(tt.err)); got != tt.want {
				t.Fatalf("DefaultErrorMapper() code = %v, want %v", got, tt.want)
			}
		})
	}
}
