// Package httperror maps project errors to HTTP response metadata.
package httperror

import (
	"net/http"

	"goshop/gmicro/errcode"
	apperrors "goshop/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Response contains the public error metadata for an HTTP response.
type Response struct {
	Status  int
	Code    int
	Message string
}

// ResponseFor returns public HTTP response metadata for err.
func ResponseFor(err error) Response {
	spec, ok := apperrors.SpecOf(err)
	if !ok || spec.Kind == "" {
		if grpcSpec, grpcOK := publicGRPCSpec(err); grpcOK {
			spec = grpcSpec
		} else {
			spec = apperrors.SpecForCode(apperrors.ParseCoder(err).Code())
		}
	}

	return Response{
		Status:  statusForKind(spec.Kind),
		Code:    spec.Code,
		Message: spec.Message,
	}
}

// publicGRPCSpec converts gRPC transport errors that reached the HTTP
// boundary without a domain adapter. Validation messages are safe only when
// authored by our services; all other categories use fixed public messages.
func publicGRPCSpec(err error) (apperrors.Spec, bool) {
	grpcStatus, ok := status.FromError(err)
	if !ok {
		return apperrors.Spec{}, false
	}

	spec := apperrors.Spec{}
	switch grpcStatus.Code() {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		message := grpcStatus.Message()
		if message == "" {
			message = "Validation failed"
		}
		spec = apperrors.Spec{Code: errcode.ErrValidation, Kind: apperrors.KindInvalidArgument, Message: message}
	case codes.NotFound:
		spec = apperrors.SpecForCode(errcode.ErrPageNotFound)
	case codes.Aborted, codes.AlreadyExists:
		spec = apperrors.SpecForCode(errcode.ErrConflict)
	case codes.PermissionDenied:
		spec = apperrors.SpecForCode(errcode.ErrPermissionDenied)
	case codes.Unauthenticated:
		spec = apperrors.SpecForCode(errcode.ErrTokenInvalid)
	case codes.DeadlineExceeded:
		spec = apperrors.SpecForCode(errcode.ErrTimeout)
	case codes.Unavailable:
		spec = apperrors.SpecForCode(errcode.ErrServiceUnavailable)
	default:
		return apperrors.Spec{}, false
	}
	return spec, true
}

func statusForKind(kind apperrors.Kind) int {
	switch kind {
	case apperrors.KindInvalidArgument:
		return http.StatusBadRequest
	case apperrors.KindUnauthenticated:
		return http.StatusUnauthorized
	case apperrors.KindPermissionDenied:
		return http.StatusForbidden
	case apperrors.KindNotFound:
		return http.StatusNotFound
	case apperrors.KindConflict:
		return http.StatusConflict
	case apperrors.KindRateLimited:
		return http.StatusTooManyRequests
	case apperrors.KindUnavailable:
		return http.StatusServiceUnavailable
	case apperrors.KindTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
