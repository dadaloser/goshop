package controller

import (
	"net/http"
	"strings"

	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authz"

	"github.com/gin-gonic/gin"
)

type createStaffUserRequest struct {
	Username string   `json:"username"`
	Mobile   string   `json:"mobile" binding:"required"`
	Email    string   `json:"email"`
	NickName string   `json:"nick_name"`
	Password string   `json:"password" binding:"required"`
	Roles    []string `json:"roles" binding:"required"`
	Status   string   `json:"status"`
}

func (us *userServer) CreateStaff(ctx *gin.Context) {
	if us == nil || us.users == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "user rpc client is not initialized",
		})
		return
	}

	var request createStaffUserRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": http.StatusBadRequest,
			"msg":  "invalid request",
		})
		return
	}
	actor, ok := currentActor(ctx)
	if !ok {
		return
	}
	if !authz.CanManageRoleSet(currentRoles(ctx), request.Roles) {
		ctx.JSON(http.StatusForbidden, gin.H{
			"code": http.StatusForbidden,
			"msg":  "cross-domain role assignment denied",
		})
		return
	}
	if !hasCurrentRole(ctx, authz.StaffRoleSuperAdmin) {
		for _, role := range request.Roles {
			if strings.EqualFold(role, string(authz.StaffRoleSuperAdmin)) {
				ctx.JSON(http.StatusForbidden, gin.H{
					"code": http.StatusForbidden,
					"msg":  "super admin role can only be assigned by super admin",
				})
				return
			}
		}
	}

	response, err := us.users.CreateStaffUser(ctx.Request.Context(), &upbv1.CreateStaffUserRequest{
		User: &upbv1.CreateUserInfo{
			Username: request.Username,
			Mobile:   request.Mobile,
			Email:    request.Email,
			NickName: request.NickName,
			PassWord: request.Password,
		},
		Roles:  request.Roles,
		Status: request.Status,
		Actor:  actor,
	})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "create staff user failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user":        response.GetUser(),
		"roles":       response.GetRoles(),
		"permissions": response.GetPermissions(),
	})
}
