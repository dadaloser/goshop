package user

import (
	"encoding/json"
	stderrors "errors"
	"goshop/app/pkg/bizcode"
	"io"
	"strings"
	"time"

	"goshop/gmicro/errcode"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/common/core"
	jtime "goshop/pkg/common/time"
	pkgerrors "goshop/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type UpdateUserForm struct {
	Name     string `form:"name" json:"name" binding:"required,min=3,max=10"`
	Gender   string `form:"gender" json:"gender" binding:"required,oneof=unknown female male"`
	Birthday string `form:"birthday" json:"birthday" binding:"required,datetime=2006-01-02"`
	Email    string `form:"email" json:"email" binding:"omitempty,email"`
}

func (us *userServer) UpdateUser(ctx *gin.Context) {
	updateForm := UpdateUserForm{}
	if err := ctx.ShouldBind(&updateForm); err != nil {
		core.WriteResponse(ctx, newUpdateValidationError(updateUserValidationMessage(err)), nil)
		return
	}

	userIDInt, err := userIDFromContext(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	userSrv, err := us.usersService()
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	userDTO, err := userSrv.Get(ctx, userIDInt)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if userDTO == nil {
		core.WriteResponse(ctx, pkgerrors.NewCode(bizcode.ErrConnectGRPC, "user service response is empty"), nil)
		return
	}
	userDTO.NickName = updateForm.Name

	//将前端传递过来的日期格式转换成int
	loc, err := time.LoadLocation("Local") //local的L必须大写
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	birthDay, err := time.ParseInLocation("2006-01-02", updateForm.Birthday, loc)
	if err != nil {
		core.WriteResponse(ctx, newUpdateValidationError("生日格式应为 YYYY-MM-DD"), nil)
		return
	}
	userDTO.NickName = updateForm.Name
	userDTO.Birthday = jtime.Time{Time: birthDay}
	userDTO.Gender = updateForm.Gender
	userDTO.Email = updateForm.Email
	err = userSrv.Update(ctx, userDTO)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"msg": "修改成功"})
}

func newUpdateValidationError(message string) error {
	return pkgerrors.WrapSpec(stderrors.New(message), pkgerrors.Spec{
		Code:    errcode.ErrValidation,
		Kind:    pkgerrors.KindInvalidArgument,
		Message: message,
	}, message)
}

func updateUserValidationMessage(err error) string {
	var validationErrors validator.ValidationErrors
	if stderrors.As(err, &validationErrors) && len(validationErrors) > 0 {
		return updateUserFieldValidationMessage(strings.ToLower(validationErrors[0].Field()))
	}

	var typeError *json.UnmarshalTypeError
	if stderrors.As(err, &typeError) {
		return updateUserFieldValidationMessage(strings.ToLower(typeError.Field))
	}
	if stderrors.Is(err, io.EOF) {
		return "请求内容不能为空"
	}
	return "请求内容格式不正确"
}

func updateUserFieldValidationMessage(field string) string {
	switch field {
	case "name":
		return "昵称长度应为 3 到 10 个字符"
	case "gender":
		return "性别只能为 unknown、female 或 male"
	case "birthday":
		return "生日格式应为 YYYY-MM-DD"
	case "email":
		return "邮箱格式不正确"
	default:
		return "请求字段格式不正确"
	}
}

func userIDFromContext(ctx *gin.Context) (uint64, error) {
	userID, ok := ctx.Get(middlewares.KeyUserID)
	if !ok {
		return 0, pkgerrors.NewCode(errcode.ErrInvalidAuthHeader, "user id is missing")
	}
	userIDFloat, ok := userID.(float64)
	if !ok {
		return 0, pkgerrors.NewCode(errcode.ErrInvalidAuthHeader, "user id has invalid type")
	}
	return uint64(userIDFloat), nil
}
