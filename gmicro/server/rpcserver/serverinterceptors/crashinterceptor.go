package serverinterceptors

import (
	"context"
	"goshop/gmicro/core/metric"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"goshop/pkg/log"
)

var metricServerPanicTotal = metric.NewCounterVec(&metric.CounterVecOpts{
	Namespace: serverNamespace,
	Subsystem: "requests",
	Name:      "goshop_panic_total",
	Help:      "rpc server panic count recovered by crash interceptor.",
	Labels:    []string{"method"},
})

func StreamCrashInterceptor(svr interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) (err error) {
	defer handleCrash(func(r interface{}) {
		metricServerPanicTotal.Inc(info.FullMethod)
		log.Errorf("%+v\n \n %s", r, debug.Stack())
		err = status.Error(codes.Internal, "internal server error")
	})

	return handler(svr, stream)
}

// UnaryCrashInterceptor 实现接口 grpc.UnaryServerInterceptor
func UnaryCrashInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer handleCrash(func(r interface{}) {
		metricServerPanicTotal.Inc(info.FullMethod)
		log.Errorf("%+v\n \n %s", r, debug.Stack())
		resp = nil
		err = status.Error(codes.Internal, "internal server error")
	})

	return handler(ctx, req)
}

func handleCrash(hanlder func(interface{})) {
	if r := recover(); r != nil {
		hanlder(r)
	}
}
