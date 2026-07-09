package user

import (
	gin2 "goshop/app/pkg/translator/gin"
	"goshop/pkg/common/core"

	"github.com/gin-gonic/gin"
)

type DeleteAccountForm struct {
	Password string `form:"password" json:"password" binding:"required,min=1,max=72"`
}

func (us *userServer) DeleteAccount(ctx *gin.Context) {
	form := DeleteAccountForm{}
	if err := ctx.ShouldBind(&form); err != nil {
		gin2.HandleValidatorError(ctx, err, us.trans)
		return
	}

	userID, err := userIDFromContext(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	userSrv, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if err = userSrv.DeleteAccount(ctx, userID, form.Password); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"ok": true})
}
