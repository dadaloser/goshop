package serverinterceptors

import (
	"context"
	"goshop/gmicro/core/metric"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var metricServerTimeoutTotal = metric.NewCounterVec(&metric.CounterVecOpts{
	Namespace: serverNamespace,
	Subsystem: "requests",
	Name:      "timeout_total",
	Help:      "rpc server requests that exceeded configured timeout.",
	Labels:    []string{"method"},
})

// UnaryTimeoutInterceptor returns a func that sets timeout to incoming unary requests.
func UnaryTimeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		resp, err := handler(ctx, req)
		if ctxErr := ctx.Err(); ctxErr != nil {
			if ctxErr == context.DeadlineExceeded {
				metricServerTimeoutTotal.Inc(info.FullMethod)
			}
			return nil, status.FromContextError(ctxErr).Err()
		}
		return resp, err
	}
}
