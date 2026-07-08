package user

import (
	"goshop/app/pkg/code"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

func (us *userServer) GetUserDetail(ctx *gin.Context) {
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
	userDTO, err := userSrv.Get(ctx, userID)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	if userDTO == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "user service response is empty"), nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{
		"name":     userDTO.NickName,
		"birthday": userDTO.Birthday.Format("2006-01-02"),
		"gender":   userDTO.Gender,
		"mobile":   userDTO.Mobile,
		"email":    userDTO.Email,
	})
}
