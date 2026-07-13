package serverinterceptors

import (
	"context"
	"goshop/gmicro/core/metric"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"goshop/pkg/log"
)

var metricServerTimeoutTotal = metric.NewCounterVec(&metric.CounterVecOpts{
	Namespace: serverNamespace,
	Subsystem: "requests",
	Name:      "goshop_timeout_total",
	Help:      "rpc server requests that exceeded configured timeout.",
	Labels:    []string{"method"},
})

// UnaryTimeoutInterceptor returns a func that sets timeout to incoming unary requests.
func UnaryTimeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		type result struct {
			resp  interface{}
			err   error
			panic interface{}
			stack []byte
		}

		done := make(chan result, 1)
		go func() {
			out := result{}
			defer func() {
				if r := recover(); r != nil {
					out.panic = r
					out.stack = debug.Stack()
				}
				done <- out
			}()

			out.resp, out.err = handler(ctx, req)
		}()

		select {
		case out := <-done:
			if out.panic != nil {
				metricServerPanicTotal.Inc(info.FullMethod)
				log.Errorf("%+v\n \n %s", out.panic, out.stack)
				return nil, status.Error(codes.Internal, "internal server error")
			}
			if err := ctx.Err(); err != nil {
				if err == context.DeadlineExceeded {
					metricServerTimeoutTotal.Inc(info.FullMethod)
				}
				return nil, status.FromContextError(err).Err()
			}
			return out.resp, out.err
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				metricServerTimeoutTotal.Inc(info.FullMethod)
			}
			return nil, status.FromContextError(ctx.Err()).Err()
		}
	}
}
