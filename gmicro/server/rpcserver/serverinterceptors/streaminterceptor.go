package serverinterceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *contextServerStream) Context() context.Context { return s.ctx }

// StreamTimeoutInterceptor bounds the total lifetime of an inbound stream.
// Stream handlers must honor stream.Context cancellation for prompt cleanup.
func StreamTimeoutInterceptor(timeout time.Duration) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, cancel := context.WithTimeout(stream.Context(), timeout)
		defer cancel()
		err := handler(srv, &contextServerStream{ServerStream: stream, ctx: ctx})
		if ctx.Err() != nil {
			return streamContextError(info.FullMethod, ctx.Err())
		}
		return err
	}
}

func streamContextError(method string, err error) error {
	if err == context.DeadlineExceeded {
		metricServerTimeoutTotal.Inc(method)
	}
	return status.FromContextError(err).Err()
}

// StreamConcurrencyInterceptor provides application-level bulkhead isolation
// for long-lived streams. Rejected streams are safe to retry at the caller's
// discretion because the handler has not started.
func StreamConcurrencyInterceptor(maxConcurrent int) grpc.StreamServerInterceptor {
	semaphore := make(chan struct{}, maxConcurrent)
	return func(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
			return handler(srv, stream)
		default:
			return status.Error(codes.ResourceExhausted, "maximum concurrent streams exceeded")
		}
	}
}
