package user

import (
	gin2 "goshop/app/pkg/translator/gin"
	"goshop/gmicro/code"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

type RefreshForm struct {
	SessionID    string `json:"session_id" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (us *userServer) Refresh(ctx *gin.Context) {
	var form RefreshForm
	if err := ctx.ShouldBindJSON(&form); err != nil {
		gin2.HandleValidatorError(ctx, err, us.trans)
		return
	}
	userSrv, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	user, err := userSrv.Refresh(ctx, form.SessionID, form.RefreshToken)
	if err != nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrTokenInvalid, "refresh token invalid"), nil)
		return
	}
	writeLoginResponse(ctx, user)
}
