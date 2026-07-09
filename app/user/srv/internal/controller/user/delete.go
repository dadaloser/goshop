package user

import (
	"context"

	upbv1 "goshop/api/user/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	"google.golang.org/protobuf/types/known/emptypb"
)

func (u *userServer) DeleteUser(ctx context.Context, request *upbv1.IdRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "delete user request is required")
	}

	log.Infof("delete user function called.")
	if err := u.srv.Delete(ctx, uint64(request.Id)); err != nil {
		log.Errorf("delete user failed: id=%d error=%v", request.Id, err)
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
