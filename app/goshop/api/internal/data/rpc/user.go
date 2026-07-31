package rpc

import (
	"context"
	"crypto/sha256"
	"goshop/app/pkg/authz"
	"goshop/app/pkg/bizcode"
	"goshop/gmicro/errcode"
	"goshop/pkg/errors"
	"goshop/pkg/transport/httperror"
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
	clientIP, location := sessionClientMetadata(ctx)
	resp, err := u.uc.CreateSession(ctx, &upbv1.CreateSessionRequest{UserId: int32(userID), DeviceId: deviceID, DeviceName: deviceName, ClientIp: clientIP, Location: location, RefreshTokenHash: hash[:], ExpiresAt: uint64(expiresAt.Unix()), PrincipalType: string(authz.PrincipalCustomer)})
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

func (u *users) ListUserSessions(ctx context.Context, userID uint64, page, pageSize int) (data.SessionList, error) {
	resp, err := u.uc.ListUserSessions(ctx, &upbv1.ListUserSessionsRequest{UserId: int32(userID), Pn: uint32(page), PSize: uint32(pageSize)})
	if err != nil {
		return data.SessionList{}, err
	}
	result := data.SessionList{TotalCount: int64(resp.GetTotal()), Items: make([]data.Session, 0, len(resp.GetItems()))}
	for _, item := range resp.GetItems() {
		session := data.Session{ID: item.GetId(), UserID: userID, DeviceID: item.GetDeviceId(), DeviceName: item.GetDeviceName(), ClientIP: item.GetClientIp(), Location: item.GetLocation(), CreatedAt: time.Unix(int64(item.GetCreatedAt()), 0), LastUsedAt: time.Unix(int64(item.GetLastUsedAt()), 0), ExpiresAt: time.Unix(int64(item.GetExpiresAt()), 0), Active: item.GetActive()}
		if item.GetRevokedAt() != 0 {
			revokedAt := time.Unix(int64(item.GetRevokedAt()), 0)
			session.RevokedAt = &revokedAt
		}
		result.Items = append(result.Items, session)
	}
	return result, nil
}

func (u *users) ListDeviceBlacklist(ctx context.Context, page, pageSize int) (data.DeviceBlacklistList, error) {
	resp, err := u.uc.ListDeviceBlacklist(ctx, &upbv1.ListDeviceBlacklistRequest{Pn: uint32(page), PSize: uint32(pageSize)})
	if err != nil {
		return data.DeviceBlacklistList{}, err
	}
	result := data.DeviceBlacklistList{TotalCount: int64(resp.GetTotal()), Items: make([]data.DeviceBlacklist, 0, len(resp.GetItems()))}
	for _, item := range resp.GetItems() {
		result.Items = append(result.Items, data.DeviceBlacklist{UserID: uint64(item.GetUserId()), DeviceID: item.GetDeviceId(), CreatedAt: time.Unix(int64(item.GetCreatedAt()), 0)})
	}
	return result, nil
}
func (u *users) AddDeviceBlacklist(ctx context.Context, userID uint64, deviceID string) error {
	_, err := u.uc.AddDeviceBlacklist(ctx, &upbv1.DeviceBlacklistRequest{UserId: int32(userID), DeviceId: deviceID})
	return err
}
func (u *users) DeleteDeviceBlacklist(ctx context.Context, userID uint64, deviceID string) error {
	_, err := u.uc.DeleteDeviceBlacklist(ctx, &upbv1.DeviceBlacklistRequest{UserId: int32(userID), DeviceId: deviceID})
	return err
}

func sessionClientMetadata(ctx context.Context) (string, string) {
	headers, ok := ctx.(interface{ GetHeader(string) string })
	if !ok {
		return "", ""
	}
	clientIP := strings.TrimSpace(headers.GetHeader("X-Real-IP"))
	if forwarded := strings.TrimSpace(headers.GetHeader("X-Forwarded-For")); forwarded != "" {
		clientIP = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	location := strings.TrimSpace(headers.GetHeader("X-Device-Location"))
	if len(clientIP) > 45 {
		clientIP = clientIP[:45]
	}
	if len(location) > 255 {
		location = location[:255]
	}
	return clientIP, location
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
		return errors.NewCode(bizcode.ErrUserPasswordIncorrect, "密码错误")
	}

	cres, err := u.uc.CheckPassWord(ctx, &upbv1.PasswordCheckInfo{
		Password:          password,
		EncryptedPassword: encryptedPwd,
	})
	if err != nil {
		return err
	}
	if cres == nil {
		return errors.NewCode(bizcode.ErrUserPasswordIncorrect, "密码错误")
	}
	if cres.Success {
		return nil
	}
	return errors.NewCode(bizcode.ErrUserPasswordIncorrect, "密码错误")
}

func (u *users) Create(ctx context.Context, user *data.UserCreate) (data.User, error) {
	if user == nil {
		return data.User{}, errcode.NewValidationError("用户信息不能为空")
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
		return data.User{}, userRPCError(err, errcode.ErrValidation)
	}
	if userRsp == nil {
		return data.User{}, errors.NewCode(bizcode.ErrConnectGRPC, "用户服务未返回创建结果")
	}
	return publicUserFromResponse(userRsp), nil
}

func (u *users) Update(ctx context.Context, user *data.User) error {
	if user == nil || user.ID == 0 {
		return errcode.NewValidationError("用户信息不能为空")
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
		return errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}

	_, err := u.uc.DeleteUser(ctx, &upbv1.IdRequest{
		Id: int32(userID),
	})
	if err != nil {
		return userRPCError(err, bizcode.ErrUserNotFound)
	}
	return nil
}

func (u *users) Get(ctx context.Context, userID uint64) (data.User, error) {
	if userID == 0 {
		return data.User{}, errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}

	user, err := u.uc.GetUserById(ctx, &upbv1.IdRequest{
		Id: int32(userID),
	})
	if err != nil {
		return data.User{}, userRPCError(err, bizcode.ErrUserNotFound)
	}
	if user == nil {
		return data.User{}, errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}

	return publicUserFromResponse(user), nil
}

func (u *users) GetByMobile(ctx context.Context, mobile string) (data.User, error) {
	return u.GetByUsername(ctx, mobile)
}

func (u *users) GetByUsername(ctx context.Context, username string) (data.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" {
		return data.User{}, errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}

	user, err := u.uc.GetUserByMobile(ctx, &upbv1.MobileRequest{
		Mobile: username,
	})
	if err != nil {
		return data.User{}, userRPCError(err, bizcode.ErrUserNotFound)
	}
	if user == nil {
		return data.User{}, errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}

	return publicUserFromResponse(user), nil
}

var _ data.UserData = &users{}

func (u *users) GetAuth(ctx context.Context, userID uint64) (data.UserAuth, error) {
	if userID == 0 {
		return data.UserAuth{}, errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}

	user, err := u.uc.GetUserAuthById(ctx, &upbv1.IdRequest{Id: int32(userID)})
	if err != nil {
		return data.UserAuth{}, userRPCError(err, bizcode.ErrUserNotFound)
	}
	if user == nil {
		return data.UserAuth{}, errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}
	return authUserFromResponse(user), nil
}

func (u *users) GetAuthByUsername(ctx context.Context, username string) (data.UserAuth, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" {
		return data.UserAuth{}, errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}

	user, err := u.uc.GetUserAuthByMobile(ctx, &upbv1.MobileRequest{Mobile: username})
	if err != nil {
		return data.UserAuth{}, userRPCError(err, bizcode.ErrUserNotFound)
	}
	if user == nil {
		return data.UserAuth{}, errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
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
	if spec, ok := httperror.SpecFromGRPC(err); ok {
		if spec.Code == errcode.ErrValidation {
			return errors.NewPublicSpec(spec, "user service validation failed")
		}
		return errors.NewSpec(spec, "user service request failed")
	}
	switch status.Code(err) {
	case codes.NotFound:
		return errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	case codes.Aborted, codes.AlreadyExists:
		return errors.NewCode(bizcode.ErrUserAlreadyExists, "user service reported a duplicate user")
	case codes.InvalidArgument:
		message := status.Convert(err).Message()
		if invalidArgumentCode == errcode.ErrValidation && message != "" {
			return errcode.NewValidationError(message)
		}
		return errors.NewCode(invalidArgumentCode, message)
	default:
		return err
	}
}
