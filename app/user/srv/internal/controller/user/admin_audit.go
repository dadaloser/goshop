package user

import (
	"context"
	"time"

	upbv1 "goshop/api/user/v1"
	v1 "goshop/app/user/srv/internal/service/v1"
	code2 "goshop/gmicro/code"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"

	"google.golang.org/protobuf/types/known/emptypb"
)

func (u *userServer) CreateAdminAuditLog(ctx context.Context, request *upbv1.CreateAdminAuditLogRequest) (*emptypb.Empty, error) {
	if request == nil || request.Log == nil {
		return nil, errors.WithCode(code2.ErrValidation, "create admin audit log request is required")
	}
	if err := u.srv.CreateAdminAuditLog(ctx, v1.AdminAuditLogDTO{
		TargetUserID:       request.Log.TargetUserId,
		ActorUserID:        request.Log.ActorUserId,
		ActorPrincipalType: request.Log.ActorPrincipalType,
		Action:             request.Log.Action,
		Detail:             request.Log.Detail,
		CorrelationID:      request.Log.CorrelationId, RequestID: request.Log.RequestId, TargetType: request.Log.TargetType, TargetID: request.Log.TargetId, Domain: request.Log.Domain, StoreID: request.Log.StoreId, TeamID: request.Log.TeamId,
	}); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (u *userServer) ListAdminAuditLogs(ctx context.Context, request *upbv1.AdminAuditLogPageRequest) (*upbv1.AdminAuditLogListResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "admin audit log request is required")
	}

	filters := v1.AdminAuditLogFilterDTO{
		TargetUserID:       request.TargetUserId,
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

	logs, err := u.srv.ListAdminAuditLogs(ctx, filters, metav1.ListMeta{
		Page:     int(request.Pn),
		PageSize: int(request.PSize),
	})
	if err != nil {
		return nil, err
	}

	response := &upbv1.AdminAuditLogListResponse{Total: int32(logs.TotalCount)}
	for _, item := range logs.Items {
		if item == nil {
			continue
		}
		response.Data = append(response.Data, &upbv1.AdminAuditLog{
			Id:                 int64(item.ID),
			TargetUserId:       item.TargetUserID,
			ActorUserId:        item.ActorUserID,
			ActorPrincipalType: item.ActorPrincipalType,
			Action:             item.Action,
			CorrelationId:      item.CorrelationID, RequestId: item.RequestID, TargetType: item.TargetType, TargetId: item.TargetID, Domain: item.Domain, StoreId: item.StoreID, TeamId: item.TeamID,
			Detail:    item.Detail,
			CreatedAt: uint64(item.CreatedAt.Unix()),
		})
	}
	return response, nil
}
