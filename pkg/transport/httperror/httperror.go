// Package httperror maps project errors to HTTP response metadata.
package httperror

import (
	"net/http"
	"strconv"

	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
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
		if grpcSpec, grpcOK := SpecFromGRPC(err); grpcOK {
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

// SpecFromGRPC restores a public business specification carried by a trusted
// GOSHOP_BUSINESS_ERROR detail. Unmarked gRPC errors use only generic catalog
// contracts and never expose their transport message.
func SpecFromGRPC(err error) (apperrors.Spec, bool) {
	grpcStatus, ok := status.FromError(err)
	if !ok {
		return apperrors.Spec{}, false
	}
	for _, detail := range grpcStatus.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok || info.GetDomain() != "goshop" || info.GetReason() != "GOSHOP_BUSINESS_ERROR" {
			continue
		}
		code, parseErr := strconv.Atoi(info.GetMetadata()["business_code"])
		if parseErr != nil || code <= 0 {
			break
		}
		spec := apperrors.SpecForCode(code)
		if spec.Code == 1 {
			break
		}
		if spec.Code == errcode.ErrValidation {
			if message := info.GetMetadata()["public_message"]; message != "" {
				spec.Message = message
			}
		}
		return spec, true
	}

	switch grpcStatus.Code() {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return apperrors.SpecForCode(errcode.ErrValidation), true
	case codes.NotFound:
		return apperrors.SpecForCode(errcode.ErrPageNotFound), true
	case codes.Aborted, codes.AlreadyExists:
		return apperrors.SpecForCode(errcode.ErrConflict), true
	case codes.PermissionDenied:
		return apperrors.SpecForCode(errcode.ErrPermissionDenied), true
	case codes.Unauthenticated:
		return apperrors.SpecForCode(errcode.ErrTokenInvalid), true
	case codes.DeadlineExceeded:
		return apperrors.SpecForCode(errcode.ErrTimeout), true
	case codes.Unavailable:
		return apperrors.SpecForCode(errcode.ErrServiceUnavailable), true
	default:
		return apperrors.Spec{}, false
	}
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
