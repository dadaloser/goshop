// Package grpcerror adapts goshop application errors to the gRPC wire protocol.
package grpcerror

import (
	"context"
	stderrors "errors"
	"strconv"

	apperrors "goshop/pkg/errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Map converts application errors to gRPC statuses while preserving the
// business error detail consumed by the HTTP gateway.
func Map(err error) error {
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

	spec, ok := apperrors.SpecOf(err)
	if !ok || spec.Kind == "" {
		spec = apperrors.SpecForCode(apperrors.ParseCoder(err).Code())
	}
	grpcStatus := status.New(codeForKind(spec.Kind), spec.Message)
	withDetails, detailErr := grpcStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: "GOSHOP_BUSINESS_ERROR",
		Domain: "goshop",
		Metadata: map[string]string{
			"business_code":  strconv.Itoa(spec.Code),
			"public_message": spec.Message,
		},
	})
	if detailErr != nil {
		return grpcStatus.Err()
	}
	return withDetails.Err()
}

func codeForKind(kind apperrors.Kind) codes.Code {
	switch kind {
	case apperrors.KindInvalidArgument:
		return codes.InvalidArgument
	case apperrors.KindUnauthenticated:
		return codes.Unauthenticated
	case apperrors.KindPermissionDenied:
		return codes.PermissionDenied
	case apperrors.KindNotFound:
		return codes.NotFound
	case apperrors.KindConflict:
		return codes.Aborted
	case apperrors.KindRateLimited:
		return codes.ResourceExhausted
	case apperrors.KindUnavailable:
		return codes.Unavailable
	case apperrors.KindTimeout:
		return codes.DeadlineExceeded
	default:
		return codes.Internal
	}
}
