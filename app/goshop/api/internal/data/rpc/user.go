package rpc

import (
	"context"
	"goshop/app/pkg/code"
	"goshop/gmicro/server/rpcserver"
	"goshop/gmicro/server/rpcserver/clientinterceptors"
	"goshop/pkg/errors"
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
	conn, err := rpcserver.DialInsecure(
		ctx,
		rpcserver.WithEndpoint(serviceName),
		rpcserver.WithDiscovery(r),
		rpcserver.WithConnectProbe(true),
		rpcserver.WithClientUnaryInterceptor(clientinterceptors.UnaryTracingInterceptor),
	)
	if err != nil {
		return nil, err
	}
	c := upbv1.NewUserClient(conn)
	return c, nil
}

func (u *users) CheckPassWord(ctx context.Context, password, encryptedPwd string) error {
	cres, err := u.uc.CheckPassWord(ctx, &upbv1.PasswordCheckInfo{
		Password:          password,
		EncryptedPassword: encryptedPwd,
	})
	if err != nil {
		return err
	}
	if cres.Success {
		return nil
	}
	return errors.WithCode(code.ErrUserPasswordIncorrect, "密码错误")
}

func (u *users) Create(ctx context.Context, user *data.User) error {
	protoUser := &upbv1.CreateUserInfo{
		Mobile:   user.Mobile,
		Email:    user.Email,
		NickName: user.NickName,
		PassWord: user.PassWord,
	}
	userRsp, err := u.uc.CreateUser(ctx, protoUser)
	if err != nil {
		return userRPCError(err, code.ErrUserAlreadyExists)
	}
	user.ID = uint64(userRsp.Id)
	return err
}

func (u *users) Update(ctx context.Context, user *data.User) error {
	protoUser := &upbv1.UpdateUserInfo{
		Id:       int32(user.ID),
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

func (u *users) Get(ctx context.Context, userID uint64) (data.User, error) {
	user, err := u.uc.GetUserById(ctx, &upbv1.IdRequest{
		Id: int32(userID),
	})
	if err != nil {
		return data.User{}, userRPCError(err, code.ErrUserNotFound)
	}

	return data.User{
		ID:       uint64(user.Id),
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
	user, err := u.uc.GetUserByMobile(ctx, &upbv1.MobileRequest{
		Mobile: username,
	})
	if err != nil {
		return data.User{}, userRPCError(err, code.ErrUserNotFound)
	}

	return data.User{
		ID:       uint64(user.Id),
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
