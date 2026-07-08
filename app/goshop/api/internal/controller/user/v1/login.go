package user

import (
	"net/mail"
	"regexp"
	"strings"

	"goshop/app/goshop/api/internal/captcha"
	userv1 "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/pkg/code"
	gin2 "goshop/app/pkg/translator/gin"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

type PassWordLoginForm struct {
	Username  string `form:"username" json:"username"`
	Mobile    string `form:"mobile" json:"mobile"`
	PassWord  string `form:"password" json:"password" binding:"required,min=1,max=72"`
	Captcha   string `form:"captcha" json:"captcha" binding:"required,min=5,max=5"`
	CaptchaId string `form:"captcha_id" json:"captcha_id" binding:"required"`
}

type SmsLoginForm struct {
	Mobile string `form:"mobile" json:"mobile" binding:"required,mobile"`
	Code   string `form:"code" json:"code" binding:"required,min=6,max=6"`
}

func (us *userServer) Login(ctx *gin.Context) {
	//表单验证
	passwordLoginForm := PassWordLoginForm{}
	if err := ctx.ShouldBind(&passwordLoginForm); err != nil {
		gin2.HandleValidatorError(ctx, err, us.trans)
		return
	}

	username := strings.TrimSpace(passwordLoginForm.Username)
	if username == "" {
		username = strings.TrimSpace(passwordLoginForm.Mobile)
	}
	if !isLoginUsername(username) {
		core.WriteResponse(ctx, errors.WithCode(code.ErrUserPasswordIncorrect, "用户名、手机号或邮箱格式错误"), nil)
		return
	}

	//验证码验证
	if !captcha.Verify(passwordLoginForm.CaptchaId, passwordLoginForm.Captcha, true) {
		core.WriteResponse(ctx, errors.WithCode(code.ErrCodeInCorrect, "验证码错误"), nil)
		return
	}

	userSrv, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	userDTO, err := userSrv.PasswordLogin(ctx, username, passwordLoginForm.PassWord)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	writeLoginResponse(ctx, userDTO)
}

func (us *userServer) SmsLogin(ctx *gin.Context) {
	loginForm := SmsLoginForm{}
	if err := ctx.ShouldBind(&loginForm); err != nil {
		gin2.HandleValidatorError(ctx, err, us.trans)
		return
	}

	userSrv, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	userDTO, err := userSrv.SmsLogin(ctx, loginForm.Mobile, loginForm.Code)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	writeLoginResponse(ctx, userDTO)
}

func writeLoginResponse(ctx *gin.Context, userDTO *userv1.UserDTO) {
	if userDTO == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "user service response is empty"), nil)
		return
	}

	core.WriteResponse(ctx, nil, gin.H{
		"id":         userDTO.ID,
		"username":   userDTO.Username,
		"nick_name":  userDTO.NickName,
		"mobile":     userDTO.Mobile,
		"email":      userDTO.Email,
		"token":      userDTO.Token,
		"expired_at": userDTO.ExpiresAt,
	})
}

func isLoginUsername(username string) bool {
	if username == "" {
		return false
	}
	if _, err := mail.ParseAddress(username); err == nil {
		return true
	}
	ok, _ := regexp.MatchString(`^[A-Za-z][A-Za-z0-9_]{2,31}$`, username)
	if ok {
		return true
	}
	ok, _ = regexp.MatchString(`^1([38][0-9]|14[579]|5[^4]|16[6]|7[1-35-8]|9[189])\d{8}$`, username)
	return ok
}
