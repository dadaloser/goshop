package user

import (
	"context"

	upbv1 "goshop/api/user/v1"

	"goshop/pkg/log"
)

func (u *userServer) GetUserByMobile(ctx context.Context, request *upbv1.MobileRequest) (*upbv1.UserInfoResponse, error) {
	log.Infof("get user by username function called.")
	user, err := u.srv.GetByUsername(ctx, request.Mobile)
	if err != nil {
		log.Errorf("get user by username: %s, error: %v", request.Mobile, err)
		return nil, err
	}

	userInfoRsp := DTOToResponse(*user)
	return userInfoRsp, nil
}
