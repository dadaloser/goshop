package serverinterceptors

import (
	"context"

	"goshop/pkg/errors"
	"goshop/pkg/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryErrorInterceptor converts project errors to standard gRPC status errors.
func UnaryErrorInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	resp, err := handler(ctx, req)
	return resp, toGRPCStatusError(info.FullMethod, err)
}

// StreamErrorInterceptor converts project errors to standard gRPC status errors.
func StreamErrorInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	return toGRPCStatusError(info.FullMethod, handler(srv, stream))
}

func toGRPCStatusError(method string, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}

	log.Errorf("[grpc] method=%s error=%+v", method, err)
	return errors.ToGrpcError(err)
}
