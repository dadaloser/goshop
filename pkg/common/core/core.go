package core

import (
	"net/http"

	"goshop/pkg/transport/httperror"

	"github.com/gin-gonic/gin"
)

// ErrResponse defines the return messages when an error occurred.
// Reference will be omitted if it does not exist.
// swagger:model
type ErrResponse struct {
	// Code defines the business error code.
	Code int `json:"code"`

	// Message contains the detail of this message.
	// This message is suitable to be exposed to external
	Message string `json:"msg"`

	Detail string `json:"detail,omitempty"`

	// Reference returns the reference document which maybe useful to solve this error.
	Reference string `json:"reference,omitempty"`
}

// WriteResponse writes an error or response data into the HTTP response body.
// Public error metadata is selected by the HTTP error adapter; internal error
// details must be logged by the request boundary and are not returned to clients.
func WriteResponse(c *gin.Context, err error, data interface{}) {
	if err != nil {
		response := httperror.ResponseFor(err)
		c.JSON(response.Status, ErrResponse{
			Code:      response.Code,
			Message:   response.Message,
			Reference: response.Reference,
		})

		return
	}

	c.JSON(http.StatusOK, data)
}
