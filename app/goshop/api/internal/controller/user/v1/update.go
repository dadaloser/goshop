package user

import (
	"fmt"
	"time"

	gin2 "goshop/app/pkg/translator/gin"
	gcode "goshop/gmicro/code"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/common/core"
	jtime "goshop/pkg/common/time"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

type UpdateUserForm struct {
	Name     string `form:"name" json:"name" binding:"required,min=3,max=10"`
	Gender   string `form:"gender" json:"gender" binding:"required,oneof=female male"`
	Birthday string `form:"birthday" json:"birthday" binding:"required,datetime=2006-01-02"`
	Email    string `form:"email" json:"email" binding:"omitempty,email"`
}

func (us *userServer) UpdateUser(ctx *gin.Context) {
	updateForm := UpdateUserForm{}
	if err := ctx.ShouldBind(&updateForm); err != nil {
		gin2.HandleValidatorError(ctx, err, us.trans)
		return
	}

	userIDInt, err := userIDFromContext(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	userDTO, err := us.sf.Users().Get(ctx, userIDInt)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
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
		core.WriteResponse(ctx, fmt.Errorf("parse birthday: %w", err), nil)
		return
	}
	userDTO.NickName = updateForm.Name
	userDTO.Birthday = jtime.Time{birthDay}
	userDTO.Gender = updateForm.Gender
	userDTO.Email = updateForm.Email
	err = us.sf.Users().Update(ctx, userDTO)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, nil)
}

func userIDFromContext(ctx *gin.Context) (uint64, error) {
	userID, ok := ctx.Get(middlewares.KeyUserID)
	if !ok {
		return 0, errors.WithCode(gcode.ErrInvalidAuthHeader, "user id is missing")
	}
	userIDFloat, ok := userID.(float64)
	if !ok {
		return 0, errors.WithCode(gcode.ErrInvalidAuthHeader, "user id has invalid type")
	}
	return uint64(userIDFloat), nil
}
