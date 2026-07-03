package user

import (
	"context"
	upbv1 "goshop/api/user/v1"
	srvv1 "goshop/app/user/srv/internal/service/v1"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/log"
)

func DTOToResponse(userDTO srvv1.UserDTO) *upbv1.UserInfoResponse {
	//在grpc的message中字段有默认值，你不能随便赋值nil进去，容易出错
	//这里要搞清， 哪些字段是有默认值
	userInfoRsp := upbv1.UserInfoResponse{
		Id:       userDTO.ID,
		PassWord: userDTO.Password,
		NickName: userDTO.NickName,
		Gender:   userDTO.Gender,
		Role:     int32(userDTO.Role),
		Mobile:   userDTO.Mobile,
		Email:    userDTO.Email,
	}
	if userDTO.Birthday != nil {
		userInfoRsp.BirthDay = uint64(userDTO.Birthday.Unix())
	}
	//注意:mutex不能拷贝,因此要返回值对象
	return &userInfoRsp
}

/*
controller 层依赖了service， service层依赖了data层：
controller层能否直接依赖data层： 可以的
controller依赖service并不是直接依赖了具体的struct而是依赖了interface
*/
func (us *userServer) GetUserList(ctx context.Context, info *upbv1.PageInfo) (*upbv1.UserListResponse, error) {
	log.Info("GetUserList is called")
	srvOpts := metav1.ListMeta{
		Page:     int(info.Pn),
		PageSize: int(info.PSize),
	}
	dtoList, err := us.srv.List(ctx, []string{}, srvOpts)
	if err != nil {
		return nil, err
	}

	var rsp upbv1.UserListResponse
	for _, value := range dtoList.Items {
		userRsp := DTOToResponse(*value)
		rsp.Data = append(rsp.Data, userRsp)
	}
	return &rsp, nil
}
