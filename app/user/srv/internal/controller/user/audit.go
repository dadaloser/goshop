package user

import (
	"context"
	"time"

	upbv1 "goshop/api/user/v1"
	v1 "goshop/app/user/srv/internal/service/v1"
	code2 "goshop/gmicro/code"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"
)

func (u *userServer) ListUserAuditLogs(ctx context.Context, request *upbv1.UserAuditLogPageRequest) (*upbv1.UserAuditLogListResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "user audit log request is required")
	}

	filters := v1.UserAuditLogFilterDTO{
		Action:             request.Action,
		ActorUserID:        request.ActorUserId,
		ActorPrincipalType: request.ActorPrincipalType,
	}
	if request.CreatedAfter > 0 {
		value := time.Unix(int64(request.CreatedAfter), 0)
		filters.CreatedAfter = &value
	}
	if request.CreatedBefore > 0 {
		value := time.Unix(int64(request.CreatedBefore), 0)
		filters.CreatedBefore = &value
	}

	logs, err := u.srv.ListUserAuditLogs(ctx, uint64(request.UserId), filters, metav1.ListMeta{
		Page:     int(request.Pn),
		PageSize: int(request.PSize),
	})
	if err != nil {
		return nil, err
	}

	response := &upbv1.UserAuditLogListResponse{Total: int32(logs.TotalCount)}
	for _, item := range logs.Items {
		if item == nil {
			continue
		}
		response.Data = append(response.Data, &upbv1.UserAuditLog{
			Id:                 int64(item.ID),
			UserId:             item.UserID,
			ActorUserId:        item.ActorUserID,
			ActorPrincipalType: item.ActorPrincipalType,
			Action:             item.Action,
			FromStatus:         item.FromStatus,
			ToStatus:           item.ToStatus,
			Detail:             item.Detail,
			CreatedAt:          uint64(item.CreatedAt.Unix()),
		})
	}
	return response, nil
}

func auditActorFromProto(actor *upbv1.AuditActor) v1.AuditActorDTO {
	if actor == nil {
		return v1.AuditActorDTO{}
	}
	return v1.AuditActorDTO{
		UserID:        actor.ActorUserId,
		PrincipalType: actor.PrincipalType,
	}
}
