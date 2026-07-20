package controller

import (
	"net/http"

	upbv1 "goshop/api/user/v1"

	"github.com/gin-gonic/gin"
)

type updateUserStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (us *userServer) GetByID(ctx *gin.Context) {
	userID, ok := parseUserID(ctx)
	if !ok {
		return
	}
	if us == nil || us.users == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "user rpc client is not initialized",
		})
		return
	}

	response, err := us.users.GetUserById(ctx.Request.Context(), &upbv1.IdRequest{Id: int32(userID)})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "get user failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user": response,
	})
}

func (us *userServer) UpdateStatus(ctx *gin.Context) {
	userID, ok := parseUserID(ctx)
	if !ok {
		return
	}
	if us == nil || us.users == nil || us.tokenVersions == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "user status backend is not initialized",
		})
		return
	}

	var request updateUserStatusRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": http.StatusBadRequest,
			"msg":  "invalid request",
		})
		return
	}

	response, err := us.users.UpdateUserStatus(ctx.Request.Context(), &upbv1.UpdateUserStatusRequest{
		Id:     int32(userID),
		Status: request.Status,
	})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "update user status failed",
		})
		return
	}
	if _, err = us.tokenVersions.Bump(ctx.Request.Context(), userID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"msg":  "user status token invalidation failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user": response,
		"session": gin.H{
			"invalidated": true,
		},
	})
}
