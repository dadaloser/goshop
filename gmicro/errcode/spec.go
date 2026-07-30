package errcode

import "goshop/pkg/errors"

// NewValidationError creates a validation error whose message has been
// deliberately selected as safe for API clients. Do not pass database, RPC,
// or other operational diagnostics to this function.
func NewValidationError(message string) error {
	return errors.NewPublicSpec(errors.Spec{
		Code:    ErrValidation,
		Kind:    errors.KindInvalidArgument,
		Message: message,
	}, message)
}

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
