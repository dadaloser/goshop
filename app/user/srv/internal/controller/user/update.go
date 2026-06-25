package user

import (
	"context"
	upbv1 "goshop/api/user/v1"
	"goshop/app/user/srv/internal/data/v1"
	v12 "goshop/app/user/srv/internal/service/v1"
	"goshop/pkg/log"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"
)

func (u *userServer) UpdateUser(ctx context.Context, request *upbv1.UpdateUserInfo) (*emptypb.Empty, error) {
	log.Infof("update user function called.")

	birthDay := time.Unix(int64(request.BirthDay), 0)
	userDO := v1.UserDO{
		BaseModel: v1.BaseModel{
			ID: request.Id,
		},
		NickName: request.NickName,
		Gender:   request.Gender,
		Birthday: &birthDay,
	}
	userDTO := v12.UserDTO{userDO}

	err := u.srv.Update(ctx, &userDTO)
	if err != nil {
		log.Errorf("update user: %v, error: %v", userDTO, err)
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
