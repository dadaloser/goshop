package serverinterceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryConcurrencyInterceptor provides application-level bulkhead isolation.
func UnaryConcurrencyInterceptor(maxConcurrent int) grpc.UnaryServerInterceptor {
	semaphore := make(chan struct{}, maxConcurrent)
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
			return handler(ctx, req)
		default:
			return nil, status.Error(codes.ResourceExhausted, "maximum concurrent unary requests exceeded")
		}
	}
}
