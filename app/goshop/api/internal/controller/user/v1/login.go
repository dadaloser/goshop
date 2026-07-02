package user

import (
	"goshop/app/pkg/code"
	gin2 "goshop/app/pkg/translator/gin"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

type PassWordLoginForm struct {
	Mobile    string `form:"mobile" json:"mobile" binding:"required,mobile"` //手机号码格式有规范可寻， 自定义validator
	PassWord  string `form:"password" json:"password" binding:"required,min=3,max=20"`
	Captcha   string `form:"captcha" json:"captcha" binding:"required,min=5,max=5"`
	CaptchaId string `form:"captcha_id" json:"captcha_id" binding:"required"`
}

func (us *userServer) Login(ctx *gin.Context) {
	//表单验证
	passwordLoginForm := PassWordLoginForm{}
	if err := ctx.ShouldBind(&passwordLoginForm); err != nil {
		gin2.HandleValidatorError(ctx, err, us.trans)
		return
	}

	//验证码验证
	if !store.Verify(passwordLoginForm.CaptchaId, passwordLoginForm.Captcha, true) {
		core.WriteResponse(ctx, errors.WithCode(code.ErrCodeInCorrect, "验证码错误"), nil)
		return
	}

	userDTO, err := us.sf.Users().MobileLogin(ctx, passwordLoginForm.Mobile, passwordLoginForm.PassWord)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{
		"id":         userDTO.ID,
		"nick_name":  userDTO.NickName,
		"token":      userDTO.Token,
		"expired_at": userDTO.ExpiresAt,
	})
}
