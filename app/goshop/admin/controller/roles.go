package controller

import (
	"net/http"
	"strconv"
	"strings"

	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authz"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
)

type replaceUserRolesRequest struct {
	Roles []string `json:"roles"`
}

func (us *userServer) ListStaffRoles(ctx *gin.Context) {
	if us == nil || us.users == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "user rpc client is not initialized",
		})
		return
	}

	response, err := us.users.ListStaffRoles(ctx.Request.Context(), &emptypb.Empty{})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "list staff roles failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"roles": response.GetRoles(),
	})
}

func (us *userServer) GetUserStaffRoles(ctx *gin.Context) {
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

	response, err := us.users.GetUserStaffRoles(ctx.Request.Context(), &upbv1.IdRequest{Id: int32(userID)})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "get user staff roles failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user_id":     response.GetUserId(),
		"roles":       response.GetRoles(),
		"permissions": response.GetPermissions(),
	})
}

func (us *userServer) ReplaceUserStaffRoles(ctx *gin.Context) {
	userID, ok := parseUserID(ctx)
	if !ok {
		return
	}
	if us == nil || us.users == nil || us.tokenVersions == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "role management backend is not initialized",
		})
		return
	}

	var request replaceUserRolesRequest
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
	if !hasCurrentRole(ctx, authz.StaffRoleSuperAdmin) {
		currentBinding, err := us.users.GetUserStaffRoles(ctx.Request.Context(), &upbv1.IdRequest{Id: int32(userID)})
		if err != nil {
			ctx.JSON(http.StatusBadGateway, gin.H{
				"code": http.StatusBadGateway,
				"msg":  "get user staff roles failed",
			})
			return
		}
		for _, role := range currentBinding.GetRoles() {
			if strings.EqualFold(role, string(authz.StaffRoleSuperAdmin)) {
				ctx.JSON(http.StatusForbidden, gin.H{
					"code": http.StatusForbidden,
					"msg":  "super admin roles can only be changed by super admin",
				})
				return
			}
		}
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

	response, err := us.users.ReplaceUserStaffRoles(ctx.Request.Context(), &upbv1.ReplaceUserStaffRolesRequest{
		UserId: int32(userID),
		Roles:  request.Roles,
		Actor:  actor,
	})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "replace user staff roles failed",
		})
		return
	}
	if _, err = us.tokenVersions.Bump(ctx.Request.Context(), userID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"msg":  "role update token invalidation failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user_id":     response.GetUserId(),
		"roles":       response.GetRoles(),
		"permissions": response.GetPermissions(),
		"session": gin.H{
			"invalidated": true,
		},
	})
}

func parseUserID(ctx *gin.Context) (uint64, bool) {
	value := ctx.Param("id")
	userID, err := strconv.ParseUint(value, 10, 64)
	if err != nil || userID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": http.StatusBadRequest,
			"msg":  "invalid user id",
		})
		return 0, false
	}
	return userID, true
}
