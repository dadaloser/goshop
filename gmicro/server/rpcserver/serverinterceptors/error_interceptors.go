package serverinterceptors

import (
	"context"

	"google.golang.org/grpc"
)

// UnaryErrorInterceptor preserves the historical constructor API and uses the
// transport-independent default mapper.
func UnaryErrorInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return UnaryErrorInterceptorWithMapper(nil)(ctx, req, info, handler)
}

// UnaryErrorInterceptorWithMapper maps handler errors at the gRPC transport
// boundary. A nil mapper uses DefaultErrorMapper.
func UnaryErrorInterceptorWithMapper(mapper ErrorMapper) grpc.UnaryServerInterceptor {
	if mapper == nil {
		mapper = DefaultErrorMapper
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		return resp, mapper(err)
	}
}

// StreamErrorInterceptor preserves the historical constructor API and uses the
// transport-independent default mapper.
func StreamErrorInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return StreamErrorInterceptorWithMapper(nil)(srv, stream, info, handler)
}

// StreamErrorInterceptorWithMapper maps stream handler errors at the gRPC
// transport boundary. A nil mapper uses DefaultErrorMapper.
func StreamErrorInterceptorWithMapper(mapper ErrorMapper) grpc.StreamServerInterceptor {
	if mapper == nil {
		mapper = DefaultErrorMapper
	}
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return mapper(handler(srv, stream))
	}
}
