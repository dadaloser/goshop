package clientinterceptors

import (
	"context"
	"errors"

	"goshop/gmicro/resilience"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SentinelInterceptor protects outbound unary RPCs with timeout, isolation, and circuit breaking.
func SentinelInterceptor(options *resilience.Options) (grpc.UnaryClientInterceptor, error) {
	guard, err := resilience.NewGuard("grpc", options, isGRPCDependencyError)
	if err != nil {
		return nil, err
	}

	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		callOptions ...grpc.CallOption,
	) error {
		resource := method
		if len(resource) > 0 && resource[0] == '/' {
			resource = resource[1:]
		}

		err := guard.Do(ctx, resource, func(callCtx context.Context) error {
			return invoker(callCtx, method, req, reply, cc, callOptions...)
		})
		if errors.Is(err, resilience.ErrBlocked) {
			return status.Error(codes.Unavailable, resilience.ErrBlocked.Error())
		}
		return err
	}, nil
}

func isGRPCDependencyError(err error) bool {
	switch status.Code(err) {
	case codes.Unknown, codes.DeadlineExceeded, codes.Internal, codes.Unavailable, codes.DataLoss:
		return true
	default:
		return false
	}
}
