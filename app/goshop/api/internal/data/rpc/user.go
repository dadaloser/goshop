package rpc

import (
	"context"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/gmicro/server/rpcserver"
	"goshop/pkg/errors"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	upbv1 "goshop/api/user/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/gmicro/registry"
	itime "goshop/pkg/common/time"
)

const serviceName = "discovery:///goshop-user-srv"

type users struct {
	uc upbv1.UserClient
}

func NewUsers(uc upbv1.UserClient) *users {
	return &users{uc}
}

// NewUserServiceClientContext creates a user client using ctx for the initial
// gRPC dial and discovery probe.
func NewUserServiceClientContext(ctx context.Context, r registry.Discovery) (upbv1.UserClient, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	conn, err := rpcserver.DialDiscoveryInsecure(
		ctx,
		rpcserver.WithEndpoint(serviceName),
		rpcserver.WithDiscovery(r),
	)
	if err != nil {
		return nil, err
	}
	c := upbv1.NewUserClient(conn)
	return c, nil
}

func (u *users) CheckPassWord(ctx context.Context, password, encryptedPwd string) error {
	if strings.TrimSpace(encryptedPwd) == "" {
		return errors.WithCode(code.ErrUserPasswordIncorrect, "密码错误")
	}

	cres, err := u.uc.CheckPassWord(ctx, &upbv1.PasswordCheckInfo{
		Password:          password,
		EncryptedPassword: encryptedPwd,
	})
	if err != nil {
		return err
	}
	if cres == nil {
		return errors.WithCode(code.ErrUserPasswordIncorrect, "密码错误")
	}
	if cres.Success {
		return nil
	}
	return errors.WithCode(code.ErrUserPasswordIncorrect, "密码错误")
}

func (u *users) Create(ctx context.Context, user *data.User) error {
	if user == nil {
		return errors.WithCode(code2.ErrValidation, "用户信息不能为空")
	}

	protoUser := &upbv1.CreateUserInfo{
		Username: user.Username,
		Mobile:   user.Mobile,
		Email:    user.Email,
		NickName: user.NickName,
		PassWord: user.PassWord,
	}
	userRsp, err := u.uc.CreateUser(ctx, protoUser)
	if err != nil {
		return userRPCError(err, code.ErrUserAlreadyExists)
	}
	if userRsp == nil {
		return errors.WithCode(code.ErrUserAlreadyExists, "用户创建失败")
	}
	user.ID = uint64(userRsp.Id)
	return err
}

func (u *users) Update(ctx context.Context, user *data.User) error {
	if user == nil || user.ID == 0 {
		return errors.WithCode(code2.ErrValidation, "用户信息不能为空")
	}

	protoUser := &upbv1.UpdateUserInfo{
		Id:       int32(user.ID),
		Username: user.Username,
		NickName: user.NickName,
		Gender:   user.Gender,
		BirthDay: uint64(user.Birthday.Unix()),
		Email:    user.Email,
	}
	_, err := u.uc.UpdateUser(ctx, protoUser)
	if err != nil {
		return err
	}
	return nil
}

func (u *users) Delete(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	_, err := u.uc.DeleteUser(ctx, &upbv1.IdRequest{
		Id: int32(userID),
	})
	if err != nil {
		return userRPCError(err, code.ErrUserNotFound)
	}
	return nil
}

func (u *users) Get(ctx context.Context, userID uint64) (data.User, error) {
	if userID == 0 {
		return data.User{}, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	user, err := u.uc.GetUserById(ctx, &upbv1.IdRequest{
		Id: int32(userID),
	})
	if err != nil {
		return data.User{}, userRPCError(err, code.ErrUserNotFound)
	}
	if user == nil {
		return data.User{}, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	return data.User{
		ID:       uint64(user.Id),
		Username: user.Username,
		Mobile:   user.Mobile,
		Email:    user.Email,
		NickName: user.NickName,
		Birthday: itime.Time{Time: time.Unix(int64(user.BirthDay), 0)},
		Gender:   user.Gender,
		Role:     user.Role,
		PassWord: user.PassWord,
	}, nil
}

func (u *users) GetByMobile(ctx context.Context, mobile string) (data.User, error) {
	return u.GetByUsername(ctx, mobile)
}

func (u *users) GetByUsername(ctx context.Context, username string) (data.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" {
		return data.User{}, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	user, err := u.uc.GetUserByMobile(ctx, &upbv1.MobileRequest{
		Mobile: username,
	})
	if err != nil {
		return data.User{}, userRPCError(err, code.ErrUserNotFound)
	}
	if user == nil {
		return data.User{}, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	return data.User{
		ID:       uint64(user.Id),
		Username: user.Username,
		Mobile:   user.Mobile,
		Email:    user.Email,
		NickName: user.NickName,
		Birthday: itime.Time{Time: time.Unix(int64(user.BirthDay), 0)},
		Gender:   user.Gender,
		Role:     user.Role,
		PassWord: user.PassWord,
	}, nil
}

var _ data.UserData = &users{}

func userRPCError(err error, invalidArgumentCode int) error {
	switch status.Code(err) {
	case codes.NotFound:
		return errors.WithCode(code.ErrUserNotFound, "用户不存在")
	case codes.AlreadyExists:
		return errors.WithCode(code.ErrUserAlreadyExists, "用户已经存在")
	case codes.InvalidArgument:
		return errors.WithCode(invalidArgumentCode, status.Convert(err).Message())
	default:
		return err
	}
}
