package serverinterceptors

import (
	"context"
	stderrors "errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorMapper converts application errors into transport errors. Applications
// inject their own mapper when they need a domain-specific wire protocol.
type ErrorMapper func(error) error

// DefaultErrorMapper converts only transport-independent errors. It deliberately
// does not depend on an application's domain error model.
func DefaultErrorMapper(err error) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, context.Canceled.Error())
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error())
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Error(codes.Internal, "internal server error")
}
