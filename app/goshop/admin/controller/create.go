package controller

import (
	"strings"

	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authz"
	"goshop/gmicro/errcode"
	"goshop/pkg/common/core"
	apperrors "goshop/pkg/errors"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
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
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "user service is temporarily unavailable")
		return
	}

	var request createStaffUserRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writePublicError(ctx, errcode.ErrValidation, apperrors.KindInvalidArgument, "invalid request")
		return
	}
	actor, ok := currentActor(ctx)
	if !ok {
		return
	}
	roleCatalog, err := us.users.ListStaffRoles(ctx.Request.Context(), &emptypb.Empty{})
	if err != nil {
		writeUserRPCError(ctx, err, "list staff roles failed")
		return
	}
	if !canManageRoleNamesWithCatalog(currentRoles(ctx), request.Roles, roleCatalog.GetRoles()) {
		writePublicError(ctx, errcode.ErrPermissionDenied, apperrors.KindPermissionDenied, "cross-domain role assignment denied")
		return
	}
	if !hasCurrentRole(ctx, authz.StaffRoleSuperAdmin) {
		for _, role := range request.Roles {
			if strings.EqualFold(role, string(authz.StaffRoleSuperAdmin)) {
				writePublicError(ctx, errcode.ErrPermissionDenied, apperrors.KindPermissionDenied, "super admin role can only be assigned by super admin")
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
		writeUserRPCError(ctx, err, "create staff user failed")
		return
	}

	core.WriteResponse(ctx, nil, gin.H{
		"user":        response.GetUser(),
		"roles":       response.GetRoles(),
		"permissions": response.GetPermissions(),
	})
}
