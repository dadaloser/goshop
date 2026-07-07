package user

import (
	gin2 "goshop/app/pkg/translator/gin"
	"goshop/pkg/common/core"

	"github.com/gin-gonic/gin"
)

type RegisterForm struct {
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

	userDTO, err := us.sf.Users().Register(ctx, regForm.Mobile, regForm.Email, regForm.PassWord, regForm.NickName, regForm.Code)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	writeLoginResponse(ctx, userDTO)
}
