package user

import (
	"goshop/app/goshop/api/internal/captcha"
	"goshop/app/pkg/bizcode"
	gin2 "goshop/app/pkg/translator/gin"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

/**
注册控制器
*/

type RegisterForm struct {
	Username        string `form:"username" json:"username" binding:"required,min=3,max=32"`
	Mobile          string `form:"mobile" json:"mobile" binding:"required,mobile"` //手机号码格式有规范可寻， 自定义validator
	Email           string `form:"email" json:"email" binding:"omitempty,email"`
	NickName        string `form:"nick_name" json:"nick_name" binding:"omitempty,min=2,max=20"`
	PassWord        string `form:"password" json:"password" binding:"required,min=8,max=72"`
	ConfirmPassword string `form:"confirm_password" json:"confirm_password"`
	Captcha         string `form:"captcha" json:"captcha"`
	CaptchaID       string `form:"captcha_id" json:"captcha_id"`
	SmsCode         string `form:"sms_code" json:"sms_code"`
}

func (us *userServer) Register(ctx *gin.Context) {
	regForm := RegisterForm{}
	if err := ctx.ShouldBind(&regForm); err != nil {
		gin2.HandleValidatorError(ctx, err, us.trans)
		return
	}
	if us.smsRegistrationVerificationEnabled {
		if len(regForm.SmsCode) != 6 {
			core.WriteResponse(ctx, errors.NewSpec(bizcode.SMSCodeIncorrectSpec, "sms verification code must be 6 digits"), nil)
			return
		}
	} else {
		if regForm.PassWord != regForm.ConfirmPassword {
			core.WriteResponse(ctx, errors.NewSpec(bizcode.SMSCodeIncorrectSpec, "password confirmation did not match"), nil)
			return
		}
		if regForm.CaptchaID == "" || !captcha.Verify(regForm.CaptchaID, regForm.Captcha, true) {
			core.WriteResponse(ctx, errors.NewSpec(bizcode.SMSCodeIncorrectSpec, "captcha verification failed"), nil)
			return
		}
	}

	userSrv, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	userDTO, err := userSrv.Register(ctx, regForm.Mobile, regForm.Email, regForm.Username, regForm.PassWord, regForm.NickName, regForm.SmsCode)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if userDTO == nil {
		core.WriteResponse(ctx, errors.NewSpec(bizcode.ConnectGRPCSpec, "user service response is empty"), nil)
		return
	}

	writeLoginResponse(ctx, userDTO)
}
