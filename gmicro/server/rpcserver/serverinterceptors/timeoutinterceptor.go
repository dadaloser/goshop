package serverinterceptors

import (
	"context"
	"goshop/gmicro/core/metric"
	"time"

	"google.golang.org/grpc"
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

		resp, err := handler(ctx, req)
		if ctx.Err() == context.DeadlineExceeded {
			metricServerTimeoutTotal.Inc(info.FullMethod)
		}
		return resp, err
	}
}
