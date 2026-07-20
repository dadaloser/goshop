package controller

import (
	"net/http"
	"strconv"

	upbv1 "goshop/api/user/v1"

	"github.com/gin-gonic/gin"
)

func (us *userServer) List(ctx *gin.Context) {
	if us == nil || us.users == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "user rpc client is not initialized",
		})
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
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "list users failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"total": response.GetTotal(),
		"items": response.GetData(),
	})
}
