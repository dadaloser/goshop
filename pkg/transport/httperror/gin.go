package httperror

import (
	"net/http"

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
