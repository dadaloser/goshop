package serverinterceptors

import (
	"context"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"goshop/pkg/log"
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
		type result struct {
			err   error
			panic interface{}
			stack []byte
		}
		done := make(chan result, 1)
		go func() {
			out := result{}
			defer func() {
				if recovered := recover(); recovered != nil {
					out.panic = recovered
					out.stack = debug.Stack()
				}
				done <- out
			}()
			out.err = handler(srv, &contextServerStream{ServerStream: stream, ctx: ctx})
		}()

		select {
		case out := <-done:
			if out.panic != nil {
				metricServerPanicTotal.Inc(info.FullMethod)
				log.Error("grpc stream panic recovered",
					log.String("method", info.FullMethod),
					log.Any("panic", out.panic),
					log.ByteString("stack", out.stack),
				)
				return status.Error(codes.Internal, "internal server error")
			}
			if ctx.Err() != nil {
				return streamContextError(info.FullMethod, ctx.Err())
			}
			return out.err
		case <-ctx.Done():
			return streamContextError(info.FullMethod, ctx.Err())
		}
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
