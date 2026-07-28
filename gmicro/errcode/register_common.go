package errcode

import "goshop/pkg/errors"

func init() {
	register(ErrUnknown, errors.KindInternal, "Internal server error")
	register(ErrBind, errors.KindInvalidArgument, "Error occurred while binding the request body to the struct")
	register(ErrValidation, errors.KindInvalidArgument, "Validation failed")
	register(ErrTokenInvalid, errors.KindUnauthenticated, "Token invalid")
	register(ErrPageNotFound, errors.KindNotFound, "Page not found")
}
