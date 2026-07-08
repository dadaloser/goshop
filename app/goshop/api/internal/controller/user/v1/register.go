package user

import (
	"goshop/app/pkg/code"
	gin2 "goshop/app/pkg/translator/gin"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

type RegisterForm struct {
	Username string `form:"username" json:"username" binding:"omitempty,min=3,max=32"`
	Mobile   string `form:"mobile" json:"mobile" binding:"required,mobile"` //手机号码格式有规范可寻， 自定义validator
	Email    string `form:"email" json:"email" binding:"omitempty,email"`
	NickName string `form:"nick_name" json:"nick_name" binding:"omitempty,min=2,max=20"`
	PassWord string `form:"password" json:"password" binding:"required,min=8,max=72"`
	Code     string `form:"code" json:"code" binding:"required,min=6,max=6"`
}

func (us *userServer) Register(ctx *gin.Context) {
	regForm := RegisterForm{}
	if err := ctx.ShouldBind(&regForm); err != nil {
		gin2.HandleValidatorError(ctx, err, us.trans)
		return
	}

	userSrv, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	userDTO, err := userSrv.Register(ctx, regForm.Mobile, regForm.Email, regForm.Username, regForm.PassWord, regForm.NickName, regForm.Code)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if userDTO == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "user service response is empty"), nil)
		return
	}

	writeLoginResponse(ctx, userDTO)
}
