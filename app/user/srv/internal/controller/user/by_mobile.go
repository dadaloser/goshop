package user

import (
	"context"

	upbv1 "goshop/api/user/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"goshop/pkg/log"
)

func (u *userServer) GetUserByMobile(ctx context.Context, request *upbv1.MobileRequest) (*upbv1.UserInfoResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "mobile request is required")
	}

	log.Infof("get user by username function called.")
	user, err := u.srv.GetByUsername(ctx, request.Mobile)
	if err != nil {
		log.Errorf("get user by username failed: identifier=%s error=%v", redactIdentifierForLog(request.Mobile), err)
		return nil, err
	}

	userInfoRsp := DTOToResponse(*user)
	return userInfoRsp, nil
}
