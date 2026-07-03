package user

import (
	"goshop/pkg/common/core"

	"github.com/gin-gonic/gin"
)

func (us *userServer) GetUserDetail(ctx *gin.Context) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	userDTO, err := us.sf.Users().Get(ctx, userID)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}
	core.WriteResponse(ctx, nil, gin.H{
		"name":     userDTO.NickName,
		"birthday": userDTO.Birthday.Format("2006-01-02"),
		"gender":   userDTO.Gender,
		"mobile":   userDTO.Mobile,
	})
}
