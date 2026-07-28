package user

import (
	"context"

	upbv1 "goshop/api/user/v1"
	"goshop/gmicro/errcode"
	"goshop/pkg/errors"
)

func (u *userServer) UpdateUserStatus(ctx context.Context, request *upbv1.UpdateUserStatusRequest) (*upbv1.UserInfoResponse, error) {
	if request == nil {
		return nil, errors.NewCode(errcode.ErrValidation, "update user status request is required")
	}

	user, err := u.srv.UpdateStatus(ctx, uint64(request.Id), request.Status, auditActorFromProto(request.Actor))
	if err != nil {
		return nil, err
	}
	return DTOToResponse(*user), nil
}
