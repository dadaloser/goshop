package serverinterceptors

import (
	"context"
	"runtime/debug"

	"google.golang.org/grpc"

	"goshop/pkg/log"
)

func StreamCrashInterceptor(svr interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo,
	handler grpc.StreamHandler) (err error) {
	defer handleCrash(func(r interface{}) {
		log.Errorf("%+v\n \n %s", r, debug.Stack())
	})

	return handler(svr, stream)
}

// UnaryCrashInterceptor 实现接口 grpc.UnaryServerInterceptor
func UnaryCrashInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer handleCrash(func(r interface{}) {
		log.Errorf("%+v\n \n %s", r, debug.Stack())
	})

	return handler(ctx, req)
}

func handleCrash(hanlder func(interface{})) {
	if r := recover(); r != nil {
		hanlder(r)
	}
}
