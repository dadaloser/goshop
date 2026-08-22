package controller

import (
	"strconv"

	upbv1 "goshop/api/user/v1"
	"goshop/pkg/common/core"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

func (us *userServer) List(ctx *gin.Context) {
	if us == nil || us.users == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "user service is temporarily unavailable")
		return
	}

	page := uint32(1)
	pageSize := uint32(10)
	if value := ctx.Query("pn"); value != "" {
		if parsed, err := strconv.ParseUint(value, 10, 32); err == nil && parsed > 0 {
			page = uint32(parsed)
		}
	}
	if value := ctx.Query("pSize"); value != "" {
		if parsed, err := strconv.ParseUint(value, 10, 32); err == nil && parsed > 0 {
			pageSize = uint32(parsed)
		}
	}

	response, err := us.users.GetUserList(ctx.Request.Context(), &upbv1.PageInfo{
		Pn:    page,
		PSize: pageSize,
	})
	if err != nil {
		writeUserRPCError(ctx, err, "list users failed")
		return
	}

	core.WriteResponse(ctx, nil, gin.H{
		"total": response.GetTotal(),
		"items": response.GetData(),
	})
}
