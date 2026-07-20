package user

import (
	"context"

	upbv1 "goshop/api/user/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
	"goshop/pkg/log"
)

func (u *userServer) GetUserAuthByMobile(ctx context.Context, request *upbv1.MobileRequest) (*upbv1.UserAuthResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "mobile request is required")
	}

	log.Infof("get user auth by username function called.")
	user, err := u.srv.GetAuthByUsername(ctx, request.Mobile)
	if err != nil {
		log.Errorf("get user auth by username failed: identifier=%s error=%v", redactIdentifierForLog(request.Mobile), err)
		return nil, err
	}

	return AuthDTOToResponse(*user), nil
}
