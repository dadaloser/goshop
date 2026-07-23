package user

import (
	"context"
	"net/mail"
	"strings"

	"goshop/app/goshop/api/internal/emailcode"
	userv1 "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/pkg/code"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

type SendEmailCodeForm struct {
	Email   string `json:"email" binding:"required,email"`
	Purpose string `json:"purpose" binding:"required,oneof=register login"`
}
type EmailLoginForm struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}
type EmailRegisterForm struct {
	Email    string `json:"email" binding:"required,email"`
	Mobile   string `json:"mobile" binding:"required,mobile"`
	Username string `json:"username" binding:"omitempty,min=3,max=32"`
	NickName string `json:"nick_name" binding:"omitempty,min=2,max=20"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Code     string `json:"code" binding:"required,len=6"`
}

type emailAuthService interface {
	EmailLogin(ctx context.Context, email, code string) (*userv1.UserDTO, error)
	EmailRegister(ctx context.Context, mobile, email, username, password, nickName, code string) (*userv1.UserDTO, error)
}

func SendEmailCode(sender emailcode.Sender) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var form SendEmailCodeForm
		if err := ctx.ShouldBindJSON(&form); err != nil {
			core.WriteResponse(ctx, errors.WithCode(code.ErrCodeInCorrect, "invalid email code request"), nil)
			return
		}
		if _, err := mail.ParseAddress(form.Email); err != nil {
			core.WriteResponse(ctx, errors.WithCode(code.ErrCodeInCorrect, "invalid email"), nil)
			return
		}
		if err := sender.Send(ctx, strings.ToLower(strings.TrimSpace(form.Email)), form.Purpose); err != nil {
			core.WriteResponse(ctx, errors.WithCode(code.ErrSmsVerifyLocked, "email verification temporarily unavailable"), nil)
			return
		}
		core.WriteResponse(ctx, nil, gin.H{"ok": true})
	}
}

func (us *userServer) EmailLogin(ctx *gin.Context) {
	var form EmailLoginForm
	if err := ctx.ShouldBindJSON(&form); err != nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrCodeInCorrect, "invalid email login request"), nil)
		return
	}
	base, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	service, ok := base.(emailAuthService)
	if !ok {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "email login unavailable"), nil)
		return
	}
	user, err := service.EmailLogin(ctx, form.Email, form.Code)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	writeLoginResponse(ctx, user)
}

func (us *userServer) EmailRegister(ctx *gin.Context) {
	var form EmailRegisterForm
	if err := ctx.ShouldBindJSON(&form); err != nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrCodeInCorrect, "invalid email registration request"), nil)
		return
	}
	base, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	service, ok := base.(emailAuthService)
	if !ok {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "email registration unavailable"), nil)
		return
	}
	user, err := service.EmailRegister(ctx, form.Mobile, form.Email, form.Username, form.Password, form.NickName, form.Code)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	writeLoginResponse(ctx, user)
}
