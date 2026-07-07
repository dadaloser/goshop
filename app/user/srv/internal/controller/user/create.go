package user

import (
	"context"
	upbv1 "goshop/api/user/v1"
	v12 "goshop/app/user/srv/internal/data/v1"
	"goshop/app/user/srv/internal/service/v1"
	"goshop/pkg/log"
)

// controller层应该是很薄的一层， 参数校验，日志打印，错误处理，调用service层
func (u *userServer) CreateUser(ctx context.Context, request *upbv1.CreateUserInfo) (*upbv1.UserInfoResponse, error) {
	log.Infof("create user function called.")

	userDO := v12.UserDO{
		Mobile:   request.Mobile,
		Email:    optionalString(request.Email),
		NickName: request.NickName,
		Password: request.PassWord,
	}
	userDTO := v1.UserDTO{UserDO: userDO}

	err := u.srv.Create(ctx, &userDTO)
	if err != nil {
		log.Errorf(
			"create user failed: mobile=%s email=%s error=%v",
			redactMobileForLog(userDTO.Mobile),
			redactOptionalEmailForLog(userDTO.Email),
			err,
		)
		return nil, err
	}

	userInfoRsp := DTOToResponse(userDTO)
	return userInfoRsp, nil
}
