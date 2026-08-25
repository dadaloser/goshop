package httperror

import (
	"net/http"

	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

// ErrResponse is the public JSON error envelope for Gin HTTP handlers.
// swagger:model
type ErrResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// WriteResponse writes either response data or a public error selected by the
// shared HTTP error adapter. Internal error details must not be returned.
func WriteResponse(c *gin.Context, err error, data any) {
	if err != nil {
		response := ResponseFor(err)
		c.JSON(response.Status, ErrResponse{Code: response.Code, Message: response.Message})
		return
	}
	c.JSON(http.StatusOK, data)
}

// WriteError writes a public error response using the shared HTTP adapter.
func WriteError(c *gin.Context, err error) {
	WriteResponse(c, err, nil)
}

// AbortWithError writes a public error response and stops remaining Gin handlers.
func AbortWithError(c *gin.Context, err error) {
	WriteError(c, err)
	c.Abort()
}

// AbortWithStatus adapts framework-generated HTTP statuses to the application's
// public JSON error envelope while preserving the original HTTP status.
func AbortWithStatus(c *gin.Context, status int) {
	code := errcode.ErrUnknown
	switch status {
	case http.StatusBadRequest, http.StatusRequestEntityTooLarge:
		code = errcode.ErrBind
	case http.StatusUnauthorized:
		code = errcode.ErrTokenInvalid
	case http.StatusForbidden:
		code = errcode.ErrPermissionDenied
	case http.StatusNotFound:
		code = errcode.ErrPageNotFound
	case http.StatusConflict:
		code = errcode.ErrConflict
	case http.StatusGatewayTimeout:
		code = errcode.ErrTimeout
	case http.StatusServiceUnavailable, http.StatusTooManyRequests:
		code = errcode.ErrServiceUnavailable
	}
	spec := apperrors.SpecForCode(code)
	c.JSON(status, ErrResponse{Code: spec.Code, Message: spec.Message})
	c.Abort()
}
