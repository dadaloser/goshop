// Package httperror maps project errors to HTTP response metadata.
package httperror

import (
	"net/http"

	apperrors "goshop/pkg/errors"
)

// Response contains the public error metadata for an HTTP response.
type Response struct {
	Status    int
	Code      int
	Message   string
	Reference string
}

// ResponseFor returns public HTTP response metadata for err.
func ResponseFor(err error) Response {
	spec, ok := apperrors.SpecOf(err)
	if !ok || spec.Kind == "" {
		spec = apperrors.SpecForCode(apperrors.ParseCoder(err).Code())
	}

	return Response{
		Status:    statusForKind(spec.Kind),
		Code:      spec.Code,
		Message:   spec.Message,
		Reference: spec.Reference,
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
