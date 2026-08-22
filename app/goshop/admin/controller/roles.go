package controller

import (
	"strconv"
	"strings"

	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authz"
	"goshop/pkg/common/core"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
)

type replaceUserRolesRequest struct {
	Roles []string `json:"roles"`
}

type updateStaffRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Domains     []string `json:"domains"`
}

func (us *userServer) ListStaffRoles(ctx *gin.Context) {
	if us == nil || us.users == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "user service is temporarily unavailable")
		return
	}

	response, err := us.users.ListStaffRoles(ctx.Request.Context(), &emptypb.Empty{})
	if err != nil {
		writeUserRPCError(ctx, err, "list staff roles failed")
		return
	}

	templates := roleTemplateViews(currentRoles(ctx))
	core.WriteResponse(ctx, nil, gin.H{
		"roles":     response.GetRoles(),
		"templates": templates,
	})
}

func (us *userServer) ListPermissionTemplates(ctx *gin.Context) {
	core.WriteResponse(ctx, nil, gin.H{
		"templates": roleTemplateViews(currentRoles(ctx)),
	})
}

func (us *userServer) CreateStaffRole(ctx *gin.Context) {
	if us == nil || us.users == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "user service is temporarily unavailable")
		return
	}

	var request updateStaffRoleRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writePublicError(ctx, errcode.ErrValidation, apperrors.KindInvalidArgument, "invalid request")
		return
	}
	if !canManageBusinessDomains(currentRoles(ctx), request.Domains) {
		writePublicError(ctx, errcode.ErrPermissionDenied, apperrors.KindPermissionDenied, "cross-domain role creation denied")
		return
	}
	if !canGrantPermissions(currentPermissions(ctx), request.Permissions) {
		writePublicError(ctx, errcode.ErrPermissionDenied, apperrors.KindPermissionDenied, "permission escalation denied")
		return
	}
	actor, ok := currentActor(ctx)
	if !ok {
		return
	}

	role, err := us.users.CreateStaffRole(ctx.Request.Context(), &upbv1.CreateStaffRoleRequest{
		Role: &upbv1.StaffRole{
			Name:        request.Name,
			Description: request.Description,
			Permissions: append([]string(nil), request.Permissions...),
			Domains:     append([]string(nil), request.Domains...),
		},
		Actor: actor,
	})
	if err != nil {
		writeUserRPCError(ctx, err, "create staff role failed")
		return
	}

	core.WriteResponse(ctx, nil, gin.H{
		"role": role,
	})
}

func (us *userServer) UpdateStaffRole(ctx *gin.Context) {
	roleName := strings.ToLower(strings.TrimSpace(ctx.Param("name")))
	if roleName == "" {
		writePublicError(ctx, errcode.ErrValidation, apperrors.KindInvalidArgument, "invalid role name")
		return
	}
	if us == nil || us.users == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "user service is temporarily unavailable")
		return
	}

	var request updateStaffRoleRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writePublicError(ctx, errcode.ErrValidation, apperrors.KindInvalidArgument, "invalid request")
		return
	}
	roleCatalog, err := us.users.ListStaffRoles(ctx.Request.Context(), &emptypb.Empty{})
	if err != nil {
		writeUserRPCError(ctx, err, "list staff roles failed")
		return
	}
	if !canManageRoleNamesWithCatalog(currentRoles(ctx), []string{roleName}, roleCatalog.GetRoles()) {
		writePublicError(ctx, errcode.ErrPermissionDenied, apperrors.KindPermissionDenied, "cross-domain role update denied")
		return
	}
	if !canGrantPermissions(currentPermissions(ctx), request.Permissions) {
		writePublicError(ctx, errcode.ErrPermissionDenied, apperrors.KindPermissionDenied, "permission escalation denied")
		return
	}
	actor, ok := currentActor(ctx)
	if !ok {
		return
	}

	role, err := us.users.UpdateStaffRole(ctx.Request.Context(), &upbv1.UpdateStaffRoleRequest{
		Role: &upbv1.StaffRole{
			Name:        roleName,
			Description: request.Description,
			Permissions: append([]string(nil), request.Permissions...),
			Domains:     append([]string(nil), request.Domains...),
		},
		Actor: actor,
	})
	if err != nil {
		writeUserRPCError(ctx, err, "update staff role failed")
		return
	}

	core.WriteResponse(ctx, nil, gin.H{
		"role": role,
	})
}

func (us *userServer) DeleteStaffRole(ctx *gin.Context) {
	roleName := strings.ToLower(strings.TrimSpace(ctx.Param("name")))
	if roleName == "" {
		writePublicError(ctx, errcode.ErrValidation, apperrors.KindInvalidArgument, "invalid role name")
		return
	}
	if us == nil || us.users == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "user service is temporarily unavailable")
		return
	}

	roleCatalog, err := us.users.ListStaffRoles(ctx.Request.Context(), &emptypb.Empty{})
	if err != nil {
		writeUserRPCError(ctx, err, "list staff roles failed")
		return
	}
	if !canManageRoleNamesWithCatalog(currentRoles(ctx), []string{roleName}, roleCatalog.GetRoles()) {
		writePublicError(ctx, errcode.ErrPermissionDenied, apperrors.KindPermissionDenied, "cross-domain role delete denied")
		return
	}
	actor, ok := currentActor(ctx)
	if !ok {
		return
	}

	if _, err = us.users.DeleteStaffRole(ctx.Request.Context(), &upbv1.DeleteStaffRoleRequest{
		Name:  roleName,
		Actor: actor,
	}); err != nil {
		writeUserRPCError(ctx, err, "delete staff role failed")
		return
	}

	core.WriteResponse(ctx, nil, gin.H{
		"msg": true,
	})
}

func (us *userServer) GetUserStaffRoles(ctx *gin.Context) {
	userID, ok := parseUserID(ctx)
	if !ok {
		return
	}
	if us == nil || us.users == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "user service is temporarily unavailable")
		return
	}

	response, err := us.users.GetUserStaffRoles(ctx.Request.Context(), &upbv1.IdRequest{Id: int32(userID)})
	if err != nil {
		writeUserRPCError(ctx, err, "get user staff roles failed")
		return
	}

	core.WriteResponse(ctx, nil, gin.H{
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
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "role management backend is temporarily unavailable")
		return
	}

	var request replaceUserRolesRequest
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
		currentBinding, err := us.users.GetUserStaffRoles(ctx.Request.Context(), &upbv1.IdRequest{Id: int32(userID)})
		if err != nil {
			writeUserRPCError(ctx, err, "get user staff roles failed")
			return
		}
		for _, role := range currentBinding.GetRoles() {
			if strings.EqualFold(role, string(authz.StaffRoleSuperAdmin)) {
				writePublicError(ctx, errcode.ErrPermissionDenied, apperrors.KindPermissionDenied, "super admin roles can only be changed by super admin")
				return
			}
		}
		for _, role := range request.Roles {
			if strings.EqualFold(role, string(authz.StaffRoleSuperAdmin)) {
				writePublicError(ctx, errcode.ErrPermissionDenied, apperrors.KindPermissionDenied, "super admin role can only be assigned by super admin")
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
		writeUserRPCError(ctx, err, "replace user staff roles failed")
		return
	}
	if _, err = us.tokenVersions.Bump(ctx.Request.Context(), userID); err != nil {
		writePublicError(ctx, errcode.ErrUnknown, apperrors.KindInternal, "role update token invalidation failed")
		return
	}

	core.WriteResponse(ctx, nil, gin.H{
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
		writePublicError(ctx, errcode.ErrValidation, apperrors.KindInvalidArgument, "invalid user id")
		return 0, false
	}
	return userID, true
}
