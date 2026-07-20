package user

import (
	"context"
	upbv1 "goshop/api/user/v1"
	v12 "goshop/app/user/srv/internal/data/v1"
	"goshop/app/user/srv/internal/service/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
	"goshop/pkg/log"
)

// controller层应该是很薄的一层， 参数校验，日志打印，错误处理，调用service层
func (u *userServer) CreateUser(ctx context.Context, request *upbv1.CreateUserInfo) (*upbv1.UserInfoResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "create user request is required")
	}

	log.Infof("create user function called.")

	userDO := v12.UserDO{
		Username: optionalString(request.Username),
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

	publicDTO, err := v1.PublicDTOFromMutation(&userDTO)
	if err != nil {
		return nil, err
	}
	userInfoRsp := DTOToResponse(*publicDTO)
	return userInfoRsp, nil
}

func (u *userServer) CreateStaffUser(ctx context.Context, request *upbv1.CreateStaffUserRequest) (*upbv1.StaffUserResponse, error) {
	if request == nil || request.User == nil {
		return nil, errors.WithCode(code2.ErrValidation, "create staff user request is required")
	}

	userDO := v12.UserDO{
		Username: optionalString(request.User.Username),
		Mobile:   request.User.Mobile,
		Email:    optionalString(request.User.Email),
		NickName: request.User.NickName,
		Password: request.User.PassWord,
	}
	userDTO := v1.UserDTO{UserDO: userDO}

	created, err := u.srv.CreateStaff(ctx, &userDTO, request.Roles, request.Status, auditActorFromProto(request.Actor))
	if err != nil {
		log.Errorf(
			"create staff user failed: mobile=%s email=%s error=%v",
			redactMobileForLog(userDTO.Mobile),
			redactOptionalEmailForLog(userDTO.Email),
			err,
		)
		return nil, err
	}

	return &upbv1.StaffUserResponse{
		User:        DTOToResponse(created.User),
		Roles:       append([]string(nil), created.StaffRoles...),
		Permissions: append([]string(nil), created.Permissions...),
	}, nil
}
