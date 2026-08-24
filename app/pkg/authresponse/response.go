// Package authresponse adapts framework authentication failures to the
// application's public HTTP error protocol.
package authresponse

import (
	stdErrors "errors"

	"goshop/gmicro/server/restserver/middlewares/auth"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"
	core "goshop/pkg/transport/httperror"

	"github.com/gin-gonic/gin"
)

// Write renders an authentication failure through the shared HTTP adapter.
func Write(c *gin.Context, err error) {
	code := errcode.ErrSignatureInvalid
	switch {
	case stdErrors.Is(err, auth.ErrMissingCredentials):
		code = errcode.ErrMissingHeader
	case stdErrors.Is(err, auth.ErrInvalidAuthorization):
		code = errcode.ErrInvalidAuthHeader
	case stdErrors.Is(err, auth.ErrExpiredCredentials):
		code = errcode.ErrExpired
	}
	core.WriteError(c, apperrors.NewCode(code, ""))
}
