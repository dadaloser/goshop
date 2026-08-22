package controller

import (
	"strings"

	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authz"
	"goshop/pkg/common/core"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (us *userServer) ListStaffSessions(ctx *gin.Context) {
	if us == nil || us.users == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "user service is temporarily unavailable")
		return
	}
	page := uint32(1)
	pageSize := uint32(20)
	if value := parseQueryInt32(ctx.Query("pn")); value > 0 {
		page = uint32(value)
	}
	if value := parseQueryInt32(ctx.Query("pSize")); value > 0 {
		pageSize = uint32(value)
	}
	activeOnly := true
	if raw := strings.TrimSpace(ctx.Query("active_only")); raw == "false" || raw == "0" {
		activeOnly = false
	}
	response, err := us.users.ListStaffSessions(ctx.Request.Context(), &upbv1.ListStaffSessionsRequest{
		UserId:         parseQueryInt32(ctx.Query("user_id")),
		Role:           strings.TrimSpace(ctx.Query("role")),
		CreatedAfter:   parseQueryUnix(ctx.Query("created_after")),
		CreatedBefore:  parseQueryUnix(ctx.Query("created_before")),
		LastUsedAfter:  parseQueryUnix(ctx.Query("last_used_after")),
		LastUsedBefore: parseQueryUnix(ctx.Query("last_used_before")),
		ActiveOnly:     activeOnly,
		Pn:             page,
		PSize:          pageSize,
	})
	if err != nil {
		writeUserRPCError(ctx, err, "list staff sessions failed")
		return
	}
	core.WriteResponse(ctx, nil, gin.H{"total": response.GetTotal(), "items": response.GetItems()})
}

func (us *userServer) RevokeStaffSession(ctx *gin.Context) {
	if us == nil || us.users == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "user service is temporarily unavailable")
		return
	}
	sessionID := strings.TrimSpace(ctx.Param("session_id"))
	if sessionID == "" {
		writePublicError(ctx, errcode.ErrValidation, apperrors.KindInvalidArgument, "invalid session id")
		return
	}
	actor, ok := currentActor(ctx)
	if !ok {
		return
	}
	if _, err := us.users.RevokeStaffSession(ctx.Request.Context(), &upbv1.RevokeStaffSessionRequest{SessionId: sessionID}); err != nil {
		writeUserRPCError(ctx, err, "revoke staff session failed")
		return
	}
	_, _ = us.users.CreateAdminAuditLog(ctx.Request.Context(), &upbv1.CreateAdminAuditLogRequest{Log: &upbv1.AdminAuditLog{
		ActorUserId:        actor.GetActorUserId(),
		ActorPrincipalType: actor.GetPrincipalType(),
		Action:             "staff_session_forced_logout",
		Detail:             "target_type:staff_session",
		CorrelationId:      uuid.NewString(),
		RequestId:          headerRequestID(ctx),
		TargetType:         "staff_session",
		TargetId:           sessionID,
		Domain:             string(authz.BusinessDomainPlatform),
	}})
	core.WriteResponse(ctx, nil, gin.H{"msg": "Device logged out successfully", "session_id": sessionID})
}

func (us *userServer) RevokeStaffUserSessions(ctx *gin.Context) {
	if us == nil || us.users == nil || us.tokenVersions == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, apperrors.KindUnavailable, "user session backend is temporarily unavailable")
		return
	}
	userID, ok := parseUserID(ctx)
	if !ok {
		return
	}
	actor, ok := currentActor(ctx)
	if !ok {
		return
	}
	if _, err := us.users.RevokeStaffUserSessions(ctx.Request.Context(), &upbv1.RevokeStaffUserSessionsRequest{UserId: int32(userID)}); err != nil {
		writeUserRPCError(ctx, err, "revoke staff sessions failed")
		return
	}
	if _, err := us.tokenVersions.Bump(ctx.Request.Context(), userID); err != nil {
		writePublicError(ctx, errcode.ErrUnknown, apperrors.KindInternal, "staff session token invalidation failed")
		return
	}
	_, _ = us.users.CreateAdminAuditLog(ctx.Request.Context(), &upbv1.CreateAdminAuditLogRequest{Log: &upbv1.AdminAuditLog{
		TargetUserId:       int32(userID),
		ActorUserId:        actor.GetActorUserId(),
		ActorPrincipalType: actor.GetPrincipalType(),
		Action:             "staff_user_sessions_forced_logout",
		Detail:             "target_type:staff_user",
		CorrelationId:      uuid.NewString(),
		RequestId:          headerRequestID(ctx),
		TargetType:         "staff_user",
		TargetId:           strings.TrimSpace(ctx.Param("id")),
		Domain:             string(authz.BusinessDomainPlatform),
	}})
	core.WriteResponse(ctx, nil, gin.H{"msg": true, "user_id": userID})
}

func headerRequestID(ctx *gin.Context) string {
	if ctx == nil {
		return uuid.NewString()
	}
	requestID := strings.TrimSpace(ctx.GetHeader("X-Request-ID"))
	if requestID == "" {
		requestID = uuid.NewString()
	}
	return requestID
}
