package serverinterceptors

import (
	"context"
	stderrors "errors"
	"net/http"

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

	if spec, ok := apperrors.SpecOf(err); ok && spec.Kind != "" {
		return status.Error(grpcCodeForKind(spec.Kind), spec.Message)
	}

	// Legacy WithCode errors retain their existing HTTP-status mapping until
	// callers migrate to Spec-based errors.
	coder := apperrors.ParseCoder(err)
	return status.Error(grpcCodeForHTTPStatus(coder.HTTPStatus()), coder.String())
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

func grpcCodeForHTTPStatus(statusCode int) codes.Code {
	switch statusCode {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return codes.Unavailable
	case http.StatusRequestTimeout:
		return codes.DeadlineExceeded
	default:
		return codes.Internal
	}
}
