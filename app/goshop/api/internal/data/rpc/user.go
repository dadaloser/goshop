package rpc

import (
	"context"
	"crypto/sha256"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
	"strings"
	"time"

	upbv1 "goshop/api/user/v1"
	"goshop/app/goshop/api/internal/data"
	itime "goshop/pkg/common/time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (u *users) RecordLogin(ctx context.Context, userID uint64, at time.Time) error {
	_, err := u.uc.RecordLogin(ctx, &upbv1.RecordLoginRequest{UserId: int32(userID), LoggedInAt: uint64(at.Unix())})
	return err
}

func (u *users) CreateSession(ctx context.Context, userID uint64, deviceID, deviceName, refreshToken string, expiresAt time.Time) (data.Session, error) {
	hash := sha256.Sum256([]byte(refreshToken))
	resp, err := u.uc.CreateSession(ctx, &upbv1.CreateSessionRequest{UserId: int32(userID), DeviceId: deviceID, DeviceName: deviceName, RefreshTokenHash: hash[:], ExpiresAt: uint64(expiresAt.Unix())})
	if err != nil {
		return data.Session{}, err
	}
	return sessionFromResponse(resp), nil
}

func (u *users) RefreshSession(ctx context.Context, sessionID, currentToken, nextToken string, expiresAt time.Time) (data.Session, error) {
	currentHash := sha256.Sum256([]byte(currentToken))
	nextHash := sha256.Sum256([]byte(nextToken))
	resp, err := u.uc.RefreshSession(ctx, &upbv1.RefreshSessionRequest{SessionId: sessionID, CurrentTokenHash: currentHash[:], NextTokenHash: nextHash[:], ExpiresAt: uint64(expiresAt.Unix())})
	if err != nil {
		return data.Session{}, err
	}
	return sessionFromResponse(resp), nil
}

func (u *users) RevokeSession(ctx context.Context, userID uint64, sessionID string) error {
	_, err := u.uc.RevokeSession(ctx, &upbv1.RevokeSessionRequest{UserId: int32(userID), SessionId: sessionID})
	return err
}

func (u *users) RevokeAllSessions(ctx context.Context, userID uint64) error {
	_, err := u.uc.RevokeAllSessions(ctx, &upbv1.IdRequest{Id: int32(userID)})
	return err
}

func (u *users) ValidateSession(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	resp, err := u.uc.ValidateSession(ctx, &upbv1.ValidateSessionRequest{UserId: int32(userID), SessionId: sessionID})
	if err != nil {
		return false, err
	}
	return resp.GetActive(), nil
}

func sessionFromResponse(resp *upbv1.SessionResponse) data.Session {
	if resp == nil {
		return data.Session{}
	}
	return data.Session{ID: resp.GetId(), UserID: uint64(resp.GetUserId()), DeviceID: resp.GetDeviceId(), DeviceName: resp.GetDeviceName(), ExpiresAt: time.Unix(int64(resp.GetExpiresAt()), 0)}
}

type users struct {
	uc upbv1.UserClient
}

func NewUsers(uc upbv1.UserClient) *users {
	return &users{uc}
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

func (u *users) Create(ctx context.Context, user *data.UserCreate) (data.User, error) {
	if user == nil {
		return data.User{}, errors.WithCode(code2.ErrValidation, "用户信息不能为空")
	}

	protoUser := &upbv1.CreateUserInfo{
		Username:       user.Username,
		Mobile:         user.Mobile,
		Email:          user.Email,
		NickName:       user.NickName,
		PassWord:       user.PassWord,
		MobileVerified: user.MobileVerified,
		EmailVerified:  user.EmailVerified,
	}
	userRsp, err := u.uc.CreateUser(ctx, protoUser)
	if err != nil {
		return data.User{}, userRPCError(err, code.ErrUserAlreadyExists)
	}
	if userRsp == nil {
		return data.User{}, errors.WithCode(code.ErrUserAlreadyExists, "用户创建失败")
	}
	return publicUserFromResponse(userRsp), nil
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

	return publicUserFromResponse(user), nil
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

	return publicUserFromResponse(user), nil
}

var _ data.UserData = &users{}

func (u *users) GetAuth(ctx context.Context, userID uint64) (data.UserAuth, error) {
	if userID == 0 {
		return data.UserAuth{}, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	user, err := u.uc.GetUserAuthById(ctx, &upbv1.IdRequest{Id: int32(userID)})
	if err != nil {
		return data.UserAuth{}, userRPCError(err, code.ErrUserNotFound)
	}
	if user == nil {
		return data.UserAuth{}, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}
	return authUserFromResponse(user), nil
}

func (u *users) GetAuthByUsername(ctx context.Context, username string) (data.UserAuth, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" {
		return data.UserAuth{}, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	user, err := u.uc.GetUserAuthByMobile(ctx, &upbv1.MobileRequest{Mobile: username})
	if err != nil {
		return data.UserAuth{}, userRPCError(err, code.ErrUserNotFound)
	}
	if user == nil {
		return data.UserAuth{}, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}
	return authUserFromResponse(user), nil
}

func publicUserFromResponse(user *upbv1.UserInfoResponse) data.User {
	if user == nil {
		return data.User{}
	}
	publicUser := data.User{
		ID:             uint64(user.Id),
		Username:       user.Username,
		Mobile:         user.Mobile,
		Email:          user.Email,
		NickName:       user.NickName,
		Birthday:       itime.Time{Time: time.Unix(int64(user.BirthDay), 0)},
		Gender:         user.Gender,
		Status:         user.Status,
		MobileVerified: user.MobileVerified,
		EmailVerified:  user.EmailVerified,
	}
	if user.LastLoginAt > 0 {
		lastLoginAt := itime.Time{Time: time.Unix(int64(user.LastLoginAt), 0)}
		publicUser.LastLoginAt = &lastLoginAt
	}
	return publicUser
}

func authUserFromResponse(user *upbv1.UserAuthResponse) data.UserAuth {
	if user == nil {
		return data.UserAuth{}
	}
	publicUser := publicUserFromResponse(user.User)
	publicUser.LegacyRole = user.LegacyRole
	return data.UserAuth{
		User:         publicUser,
		PasswordHash: user.PasswordHash,
		StaffRoles:   append([]string(nil), user.StaffRoles...),
		Permissions:  append([]string(nil), user.Permissions...),
	}
}

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
