package errcode

import (
	"net/http"

	"goshop/pkg/errors"
)

func init() {
	register(ErrUnknown, http.StatusInternalServerError, errors.KindInternal, "Internal server error")
	register(ErrBind, http.StatusBadRequest, errors.KindInvalidArgument, "Error occurred while binding the request body to the struct")
	register(ErrValidation, http.StatusBadRequest, errors.KindInvalidArgument, "Validation failed")
	register(ErrTokenInvalid, http.StatusUnauthorized, errors.KindUnauthenticated, "Token invalid")
	register(ErrPageNotFound, http.StatusNotFound, errors.KindNotFound, "Page not found")
}
