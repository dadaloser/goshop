package user

import (
	"context"

	upbv1 "goshop/api/user/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
	"goshop/pkg/log"
)

func (u *userServer) GetUserAuthById(ctx context.Context, request *upbv1.IdRequest) (*upbv1.UserAuthResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "user id request is required")
	}

	log.Infof("get user auth by id function called.")
	user, err := u.srv.GetAuthByID(ctx, uint64(request.Id))
	if err != nil {
		log.Errorf("get user auth by id failed: id=%d error=%v", request.Id, err)
		return nil, err
	}

	return AuthDTOToResponse(*user), nil
}
