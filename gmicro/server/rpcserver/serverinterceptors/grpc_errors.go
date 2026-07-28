package serverinterceptors

import (
	"context"
	stderrors "errors"

	apperrors "goshop/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, context.Canceled.Error())
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error())
	}

	spec, ok := apperrors.SpecOf(err)
	if !ok || spec.Kind == "" {
		spec = apperrors.SpecForCode(apperrors.ParseCoder(err).Code())
	}

	return status.Error(grpcCodeForKind(spec.Kind), spec.Message)
}

func grpcCodeForKind(kind apperrors.Kind) codes.Code {
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
