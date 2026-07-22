package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authz"

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
	actor, ok := currentActor(ctx)
	if !ok {
		return
	}
	if uint64(actor.GetActorUserId()) == userID && strings.ToLower(strings.TrimSpace(request.Status)) != string(authz.AccountStatusActive) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": http.StatusBadRequest,
			"msg":  "current staff account cannot disable itself",
		})
		return
	}
	if !hasCurrentRole(ctx, authz.StaffRoleSuperAdmin) {
		targetRoles, err := us.users.GetUserStaffRoles(ctx.Request.Context(), &upbv1.IdRequest{Id: int32(userID)})
		if err != nil {
			ctx.JSON(http.StatusBadGateway, gin.H{
				"code": http.StatusBadGateway,
				"msg":  "get user staff roles failed",
			})
			return
		}
		for _, role := range targetRoles.GetRoles() {
			if strings.EqualFold(role, string(authz.StaffRoleSuperAdmin)) {
				ctx.JSON(http.StatusForbidden, gin.H{
					"code": http.StatusForbidden,
					"msg":  "super admin status can only be changed by super admin",
				})
				return
			}
		}
	}

	response, err := us.users.UpdateUserStatus(ctx.Request.Context(), &upbv1.UpdateUserStatusRequest{
		Id:     int32(userID),
		Status: request.Status,
		Actor:  actor,
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

func (us *userServer) ListAuditLogs(ctx *gin.Context) {
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

	response, err := us.users.ListUserAuditLogs(ctx.Request.Context(), &upbv1.UserAuditLogPageRequest{
		UserId:             int32(userID),
		Pn:                 page,
		PSize:              pageSize,
		Action:             strings.TrimSpace(ctx.Query("action")),
		ActorUserId:        parseQueryInt32(ctx.Query("actor_user_id")),
		ActorPrincipalType: strings.TrimSpace(ctx.Query("actor_principal_type")),
		CreatedAfter:       parseQueryUnix(ctx.Query("created_after")),
		CreatedBefore:      parseQueryUnix(ctx.Query("created_before")),
	})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "list user audit logs failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"total": response.GetTotal(),
		"items": response.GetData(),
	})
}

func (us *userServer) ListAdminAuditLogs(ctx *gin.Context) {
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

	response, err := us.users.ListAdminAuditLogs(ctx.Request.Context(), &upbv1.AdminAuditLogPageRequest{
		TargetUserId:       parseQueryInt32(ctx.Query("target_user_id")),
		Pn:                 page,
		PSize:              pageSize,
		Action:             strings.TrimSpace(ctx.Query("action")),
		ActorUserId:        parseQueryInt32(ctx.Query("actor_user_id")),
		ActorPrincipalType: strings.TrimSpace(ctx.Query("actor_principal_type")),
		CreatedAfter:       parseQueryUnix(ctx.Query("created_after")),
		CreatedBefore:      parseQueryUnix(ctx.Query("created_before")),
	})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "list admin audit logs failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"total": response.GetTotal(),
		"items": response.GetData(),
	})
}

func parseQueryInt32(value string) int32 {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil || parsed <= 0 {
		return 0
	}
	return int32(parsed)
}

func parseQueryUnix(value string) uint64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed > 0 {
		return uint64(parsed)
	}
	if parsedTime, err := time.Parse(time.RFC3339, value); err == nil && !parsedTime.IsZero() {
		return uint64(parsedTime.Unix())
	}
	return 0
}
