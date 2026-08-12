package admin

import (
	"goshop/app/pkg/errorcatalog"
	"goshop/pkg/common/core"
	apperrors "goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

/**
Admin 根包的错误响应也统一走 core.WriteResponse。
*/

// writePublicError is the sole error response path for admin handlers.
func writePublicError(ctx *gin.Context, code int, kind apperrors.Kind, message string) {
	if ctx == nil {
		return
	}
	core.WriteResponse(ctx, apperrors.NewSpec(apperrors.Spec{Code: code, Kind: kind, Message: message}, message), nil)
}

func writeValidationError(ctx *gin.Context, message string) {
	if ctx == nil {
		return
	}
	core.WriteResponse(ctx, errorcatalog.NewValidationError(message), nil)
}
