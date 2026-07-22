package user

import (
	"context"

	upbv1 "goshop/api/user/v1"
	v1 "goshop/app/user/srv/internal/service/v1"
	code2 "goshop/gmicro/code"
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
	}); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
