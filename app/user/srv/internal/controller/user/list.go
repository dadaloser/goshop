package user

import (
	"context"
	upbv1 "goshop/api/user/v1"
	srvv1 "goshop/app/user/srv/internal/service/v1"
	code2 "goshop/gmicro/code"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"
	"goshop/pkg/log"
)

func DTOToResponse(userDTO srvv1.UserPublicDTO) *upbv1.UserInfoResponse {
	//在grpc的message中字段有默认值，你不能随便赋值nil进去，容易出错
	//这里要搞清， 哪些字段是有默认值
	userInfoRsp := upbv1.UserInfoResponse{
		Id:             userDTO.ID,
		NickName:       userDTO.NickName,
		Gender:         userDTO.Gender,
		Mobile:         userDTO.Mobile,
		Status:         userDTO.Status,
		MobileVerified: userDTO.MobileVerified,
		EmailVerified:  userDTO.EmailVerified,
	}

	//最后登陆时间
	if userDTO.LastLoginAt != nil {
		userInfoRsp.LastLoginAt = uint64(userDTO.LastLoginAt.Unix())
	}
	if userDTO.Username != nil {
		userInfoRsp.Username = *userDTO.Username
	}
	if userDTO.Email != nil {
		userInfoRsp.Email = *userDTO.Email
	}
	if userDTO.Birthday != nil {
		userInfoRsp.BirthDay = uint64(userDTO.Birthday.Unix())
	}
	//注意:mutex不能拷贝,因此要返回值对象
	return &userInfoRsp
}

func AuthDTOToResponse(userDTO srvv1.UserAuthDTO) *upbv1.UserAuthResponse {
	return &upbv1.UserAuthResponse{
		User:            DTOToResponse(userDTO.UserPublicDTO),
		PasswordHash:    userDTO.PasswordHash,
		LegacyRole:      userDTO.LegacyRole,
		StaffRoles:      append([]string(nil), userDTO.StaffRoles...),
		Permissions:     append([]string(nil), userDTO.Permissions...),
		ResourceDomains: append([]string(nil), userDTO.ResourceDomains...),
		ResourceStores:  append([]string(nil), userDTO.ResourceStores...),
		ResourceTeams:   append([]string(nil), userDTO.ResourceTeams...),
	}
}

/*
controller 层依赖了service， service层依赖了data层：
controller层能否直接依赖data层： 可以的
controller依赖service并不是直接依赖了具体的struct而是依赖了interface
*/
func (us *userServer) GetUserList(ctx context.Context, info *upbv1.PageInfo) (*upbv1.UserListResponse, error) {
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "page request is required")
	}

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

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
