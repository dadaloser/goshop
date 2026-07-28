package errcode

import "goshop/pkg/errors"

// ValidationSpec describes invalid caller input shared by RPC services.
var ValidationSpec = errors.Spec{
	Code:    ErrValidation,
	Kind:    errors.KindInvalidArgument,
	Message: "Validation failed",
}

// TokenInvalidSpec describes an invalid or expired refresh token.
var TokenInvalidSpec = errors.Spec{
	Code:    ErrTokenInvalid,
	Kind:    errors.KindUnauthenticated,
	Message: "Token invalid",
}

// UnknownSpec describes an internal error that must not expose diagnostics.
var UnknownSpec = errors.Spec{
	Code:    ErrUnknown,
	Kind:    errors.KindInternal,
	Message: "Internal server error",
}
