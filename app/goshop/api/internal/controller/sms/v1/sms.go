package sms

import (
	"goshop/app/goshop/api/internal/captcha"
	"goshop/app/goshop/api/internal/service"
	v1 "goshop/app/goshop/api/internal/service/sms/v1"
	"goshop/app/goshop/api/internal/smscode"
	"goshop/app/goshop/api/internal/smslimit"
	"goshop/app/pkg/code"
	gin2 "goshop/app/pkg/translator/gin"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type SendSmsForm struct {
	Mobile    string `form:"mobile" json:"mobile" binding:"required,mobile"` //手机号码格式有规范可寻， 自定义validator
	Type      uint   `form:"type" json:"type" binding:"required,oneof=1 2"`
	Captcha   string `form:"captcha" json:"captcha" binding:"required,min=5,max=5"`
	CaptchaId string `form:"captcha_id" json:"captcha_id" binding:"required"`
	//1. 注册发送短信验证码和动态验证码登录发送验证码
}

type SmsController struct {
	sf        service.ServiceFactory
	trans     ut.Translator
	codeStore smscode.Store
	limiter   smslimit.Store
}

func NewSmsController(sf service.ServiceFactory, trans ut.Translator, codeStore smscode.Store, limiter smslimit.Store) *SmsController {
	return &SmsController{sf: sf, trans: trans, codeStore: codeStore, limiter: limiter}
}

func (sc *SmsController) SendSms(c *gin.Context) {
	sendSmsForm := SendSmsForm{}
	if err := c.ShouldBind(&sendSmsForm); err != nil {
		gin2.HandleValidatorError(c, err, sc.trans)
		return
	}

	if !captcha.Verify(sendSmsForm.CaptchaId, sendSmsForm.Captcha, true) {
		core.WriteResponse(c, errors.WithCode(code.ErrCodeInCorrect, "验证码错误"), nil)
		return
	}

	if sc.limiter != nil {
		allowed, err := sc.limiter.Take(c.Request.Context(), sendSmsForm.Mobile, sendSmsForm.Type)
		if err != nil {
			core.WriteResponse(c, errors.WithCode(code.ErrSmsRateLimited, "短信发送暂时不可用，请稍后重试"), nil)
			return
		}
		if !allowed {
			core.WriteResponse(c, errors.WithCode(code.ErrSmsRateLimited, "短信发送过于频繁，请稍后重试"), nil)
			return
		}
	}

	smsCode, err := v1.GenerateSmsCode(6)
	if err != nil {
		core.WriteResponse(c, errors.WithCode(code.ErrSmsSend, err.Error()), nil)
		return
	}
	err = sc.sf.Sms().SendSms(c, sendSmsForm.Mobile, "SMS_181850725", "{\"code\":"+smsCode+"}")
	if err != nil {
		core.WriteResponse(c, errors.WithCode(code.ErrSmsSend, err.Error()), nil)
		return
	}

	//将验证码保存起来 - redis
	key := smscode.Key(sendSmsForm.Mobile, sendSmsForm.Type)
	if err := sc.codeStore.Set(c, key, smsCode, smscode.DefaultTTL); err != nil {
		core.WriteResponse(c, errors.WithCode(code.ErrSmsSend, err.Error()), nil)
		return
	}

	core.WriteResponse(c, nil, nil)
}
