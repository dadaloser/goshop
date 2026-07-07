package user

import (
	"context"

	upbv1 "goshop/api/user/v1"

	"goshop/pkg/log"
)

func (u *userServer) GetUserById(ctx context.Context, request *upbv1.IdRequest) (*upbv1.UserInfoResponse, error) {
	log.Infof("get user by id function called.")
	user, err := u.srv.GetByID(ctx, uint64(request.Id))
	if err != nil {
		log.Errorf("get user by id failed: id=%d error=%v", request.Id, err)
		return nil, err
	}

	userInfoRsp := DTOToResponse(*user)
	return userInfoRsp, nil
}
