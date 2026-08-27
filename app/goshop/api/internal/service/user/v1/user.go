package v1

import (
	"context"
	stderrors "errors"
	"strings"

	"goshop/app/goshop/api/internal/emailcode"
	"goshop/app/goshop/api/internal/loginattempt"
	"goshop/app/goshop/api/internal/smsattempt"
	"goshop/app/pkg/bizcode"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	"goshop/app/goshop/api/internal/data"
	"goshop/app/goshop/api/internal/smscode"
	"goshop/app/pkg/authsession/tokenversion"
	"goshop/app/pkg/options"
	"goshop/pkg/errcode"
	"goshop/pkg/storage"
)

/**
注册服务
*/

type UserDTO struct {
	data.User

	Token            string `json:"token"`
	ExpiresAt        int64  `json:"expires_at"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	RefreshExpiresAt int64  `json:"refresh_expires_at,omitempty"`
	SessionID        string `json:"session_id,omitempty"`
	LoginRequired    bool   `json:"login_required,omitempty"`
}

type UserSrv interface {
	PasswordLogin(ctx context.Context, username, password string) (*UserDTO, error)
	SmsLogin(ctx context.Context, mobile, smsCode string) (*UserDTO, error)
	Register(ctx context.Context, mobile, email, username, password, nickName, code string) (*UserDTO, error)
	Update(ctx context.Context, userDTO *UserDTO) error
	Get(ctx context.Context, userID uint64) (*UserDTO, error)
	GetByUsername(ctx context.Context, username string) (*UserDTO, error)
	LogoutAll(ctx context.Context, userID uint64) error
	Logout(ctx context.Context, userID uint64, sessionID string) error
	Refresh(ctx context.Context, sessionID, refreshToken string) (*UserDTO, error)
	ListDevices(ctx context.Context, userID uint64, page, pageSize int) (data.SessionList, error)
	LogoutDevice(ctx context.Context, userID uint64, sessionID string) error
	ListDeviceBlacklist(ctx context.Context, userID uint64, page, pageSize int) (data.DeviceBlacklistList, error)
	AddDeviceBlacklist(ctx context.Context, userID uint64, deviceID string) error
	DeleteDeviceBlacklist(ctx context.Context, userID uint64, deviceID string) error
	DeleteAccount(ctx context.Context, userID uint64, password string) error
}

type userService struct {
	//ud data.UserData
	data data.DataFactory

	jwtOpts *options.JwtOptions

	codeStore smscode.Store

	loginAttempts   loginattempt.Store
	loginIPAttempts loginattempt.Store

	smsAttempts smsattempt.Store

	smsRegistrationVerificationEnabled bool

	tokenVersions tokenversion.Store
	emailCodes    emailcode.Store
}

func NewUserService(data data.DataFactory, jwtOpts *options.JwtOptions, codeStore smscode.Store, loginAttempts loginattempt.Store, smsAttempts smsattempt.Store, tokenVersions tokenversion.Store) UserSrv {
	return NewUserServiceWithIPAttempts(data, jwtOpts, codeStore, loginAttempts, loginattempt.NewIPRedisStore(nil), smsAttempts, tokenVersions)
}

func NewUserServiceWithIPAttempts(data data.DataFactory, jwtOpts *options.JwtOptions, codeStore smscode.Store, loginAttempts, loginIPAttempts loginattempt.Store, smsAttempts smsattempt.Store, tokenVersions tokenversion.Store) UserSrv {
	return NewUserServiceWithIPAttemptsAndSMSRegistrationVerification(data, jwtOpts, codeStore, loginAttempts, loginIPAttempts, smsAttempts, tokenVersions, false)
}

// NewUserServiceWithIPAttemptsAndSMSRegistrationVerification configures
// whether registration must consume a SMS verification code.
func NewUserServiceWithIPAttemptsAndSMSRegistrationVerification(data data.DataFactory, jwtOpts *options.JwtOptions, codeStore smscode.Store, loginAttempts, loginIPAttempts loginattempt.Store, smsAttempts smsattempt.Store, tokenVersions tokenversion.Store, smsRegistrationVerificationEnabled bool) UserSrv {
	return NewUserServiceWithIPAttemptsAndSMSRegistrationVerificationAndRedis(
		data, jwtOpts, codeStore, loginAttempts, loginIPAttempts, smsAttempts, tokenVersions, smsRegistrationVerificationEnabled, nil,
	)
}

// NewUserServiceWithIPAttemptsAndSMSRegistrationVerificationAndRedis binds
// internally created Redis stores to the application-owned Redis client.
func NewUserServiceWithIPAttemptsAndSMSRegistrationVerificationAndRedis(data data.DataFactory, jwtOpts *options.JwtOptions, codeStore smscode.Store, loginAttempts, loginIPAttempts loginattempt.Store, smsAttempts smsattempt.Store, tokenVersions tokenversion.Store, smsRegistrationVerificationEnabled bool, redisClient *storage.Client) UserSrv {
	return &userService{
		data:                               data,
		jwtOpts:                            jwtOpts,
		codeStore:                          codeStore,
		loginAttempts:                      loginAttempts,
		loginIPAttempts:                    loginIPAttempts,
		smsAttempts:                        smsAttempts,
		smsRegistrationVerificationEnabled: smsRegistrationVerificationEnabled,
		tokenVersions:                      tokenVersions,
		emailCodes:                         emailcode.NewRedisStore(redisClient),
	}
}

func (us *userService) EmailLogin(ctx context.Context, email, verificationCode string) (*UserDTO, error) {
	email = normalizeLoginIdentifier(email)
	if us.emailCodes == nil {
		return nil, errors.NewSpec(bizcode.EmailVerificationUnavailableSpec, "email verification unavailable")
	}
	if err := us.emailCodes.Consume(ctx, email, "login", verificationCode); err != nil {
		return nil, errors.NewSpec(bizcode.SMSCodeIncorrectSpec, "email verification code did not match")
	}
	users, err := us.usersData()
	if err != nil {
		return nil, err
	}
	user, err := users.GetAuthByUsername(ctx, email)
	if err != nil {
		return nil, err
	}
	if !user.EmailVerified {
		return nil, errors.NewSpec(bizcode.UserAccountInactiveSpec, "email has not been verified")
	}
	token, expiresAt, refreshToken, refreshExpiresAt, sessionID, err := us.createToken(ctx, user)
	if err != nil {
		return nil, err
	}
	return &UserDTO{User: user.User, Token: token, ExpiresAt: expiresAt, RefreshToken: refreshToken, RefreshExpiresAt: refreshExpiresAt, SessionID: sessionID}, nil
}

func (us *userService) EmailRegister(ctx context.Context, mobile, email, username, password, nickName, verificationCode string) (*UserDTO, error) {
	email = normalizeLoginIdentifier(email)
	if us.emailCodes == nil {
		return nil, errors.NewSpec(bizcode.EmailVerificationUnavailableSpec, "email verification unavailable")
	}
	if err := us.emailCodes.Consume(ctx, email, "register", verificationCode); err != nil {
		return nil, errors.NewSpec(bizcode.SMSCodeIncorrectSpec, "email verification code did not match")
	}
	users, err := us.usersData()
	if err != nil {
		return nil, err
	}
	created, err := users.Create(ctx, &data.UserCreate{Username: username, Mobile: mobile, Email: email, NickName: nickName, PassWord: password, EmailVerified: true})
	if err != nil {
		return nil, err
	}
	token, expiresAt, refreshToken, refreshExpiresAt, sessionID, err := us.createToken(ctx, data.UserAuth{User: created})
	if err != nil {
		return nil, err
	}
	return &UserDTO{User: created, Token: token, ExpiresAt: expiresAt, RefreshToken: refreshToken, RefreshExpiresAt: refreshExpiresAt, SessionID: sessionID}, nil
}

func (us *userService) PasswordLogin(ctx context.Context, username, password string) (*UserDTO, error) {
	username = normalizeLoginIdentifier(username)
	clientIP := loginClientIP(ctx)
	users, err := us.usersData()
	if err != nil {
		return nil, err
	}
	if err := us.ensurePasswordLoginAllowed(ctx, username, clientIP); err != nil {
		return nil, err
	}

	user, err := users.GetAuthByUsername(ctx, username)
	if err != nil {
		if errors.IsCode(err, bizcode.ErrUserNotFound) {
			if lockedErr := us.recordPasswordLoginFailure(ctx, username, clientIP); lockedErr != nil {
				return nil, lockedErr
			}
			return nil, errors.NewSpec(bizcode.UserPasswordIncorrectSpec, "mobile number or password was incorrect")
		}
		return nil, err
	}

	//检查密码是否正确
	err = users.CheckPassWord(ctx, password, user.PasswordHash)
	if err != nil {
		if errors.IsCode(err, bizcode.ErrUserPasswordIncorrect) {
			if lockedErr := us.recordPasswordLoginFailure(ctx, username, clientIP); lockedErr != nil {
				return nil, lockedErr
			}
			return nil, errors.NewSpec(bizcode.UserPasswordIncorrectSpec, "mobile number or password was incorrect")
		}
		return nil, err
	}

	us.resetPasswordLoginFailures(ctx, username, clientIP)

	token, expiresAt, refreshToken, refreshExpiresAt, sessionID, err := us.createToken(ctx, user)
	if err != nil {
		return nil, err
	}

	return &UserDTO{
		User:         user.User,
		Token:        token,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken, RefreshExpiresAt: refreshExpiresAt, SessionID: sessionID,
	}, nil
}

func (us *userService) SmsLogin(ctx context.Context, mobile, smsCode string) (*UserDTO, error) {
	if err := us.ensureSmsCodeAllowed(ctx, mobile, smscode.TypeLogin); err != nil {
		return nil, err
	}
	if us == nil || us.codeStore == nil {
		return nil, errors.NewSpec(bizcode.ConnectGRPCSpec, "sms code store is not initialized")
	}

	key := smscode.LoginKey(mobile)
	value, err := us.codeStore.Get(ctx, key)
	if err != nil {
		if isContextError(err) {
			return nil, err
		}
		if !stderrors.Is(err, storage.ErrKeyNotFound) {
			return nil, err
		}
		if lockedErr := us.recordSmsCodeFailure(ctx, mobile, smscode.TypeLogin); lockedErr != nil {
			return nil, lockedErr
		}
		return nil, errors.NewSpec(bizcode.SMSCodeNotExistSpec, "sms verification code was not found")
	}
	if value != smsCode {
		if lockedErr := us.recordSmsCodeFailure(ctx, mobile, smscode.TypeLogin); lockedErr != nil {
			return nil, lockedErr
		}
		return nil, errors.NewSpec(bizcode.SMSCodeIncorrectSpec, "sms verification code did not match")
	}

	us.resetSmsCodeFailures(ctx, mobile, smscode.TypeLogin)

	users, err := us.usersData()
	if err != nil {
		return nil, err
	}
	user, err := users.GetAuthByUsername(ctx, mobile)
	if err != nil {
		return nil, err
	}

	if deleted, err := us.codeStore.Delete(ctx, key); err != nil {
		return nil, err
	} else if !deleted {
		return nil, errors.NewSpec(bizcode.SMSCodeNotExistSpec, "sms verification code was not found")
	}

	token, expiresAt, refreshToken, refreshExpiresAt, sessionID, err := us.createToken(ctx, user)
	if err != nil {
		return nil, err
	}
	return &UserDTO{
		User:         user.User,
		Token:        token,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken, RefreshExpiresAt: refreshExpiresAt, SessionID: sessionID,
	}, nil
}

func (us *userService) Register(ctx context.Context, mobile, email, username, password, nickName, codes string) (*UserDTO, error) {
	if us.smsRegistrationVerificationEnabled {
		if err := us.consumeRegistrationSMSCode(ctx, mobile, codes); err != nil {
			return nil, err
		}
	}

	var user = &data.UserCreate{
		Username: username,
		Mobile:   mobile,
		Email:    email,
		NickName: nickName,
		PassWord: password,
		// 只有注册短信验证码被成功消费后，才可标记手机号已验证。
		MobileVerified: us.smsRegistrationVerificationEnabled,
	}
	users, err := us.usersData()
	if err != nil {
		return nil, err
	}
	createdUser, err := users.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	token, expiresAt, refreshToken, refreshExpiresAt, sessionID, err := us.createToken(ctx, data.UserAuth{User: createdUser, PasswordHash: ""})
	if err != nil {
		log.Errorf("user %d registered but automatic login failed: %v", createdUser.ID, err)
		return &UserDTO{User: createdUser, LoginRequired: true}, nil
	}

	return &UserDTO{
		User:         createdUser,
		Token:        token,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken, RefreshExpiresAt: refreshExpiresAt, SessionID: sessionID,
	}, nil
}

func (us *userService) consumeRegistrationSMSCode(ctx context.Context, mobile, code string) error {
	if err := us.ensureSmsCodeAllowed(ctx, mobile, smscode.TypeRegister); err != nil {
		return err
	}
	if us.codeStore == nil {
		return errors.NewSpec(bizcode.ConnectGRPCSpec, "sms code store is not initialized")
	}
	if err := us.codeStore.Consume(ctx, smscode.RegisterKey(mobile), code); err != nil {
		if isContextError(err) {
			return err
		}
		if !stderrors.Is(err, storage.ErrKeyNotFound) && !stderrors.Is(err, smscode.ErrCodeMismatch) {
			return err
		}
		if lockedErr := us.recordSmsCodeFailure(ctx, mobile, smscode.TypeRegister); lockedErr != nil {
			return lockedErr
		}
		if stderrors.Is(err, storage.ErrKeyNotFound) {
			return errors.NewSpec(bizcode.SMSCodeNotExistSpec, "sms verification code was not found")
		}
		return errors.NewSpec(bizcode.SMSCodeIncorrectSpec, "registration sms code did not match")
	}
	us.resetSmsCodeFailures(ctx, mobile, smscode.TypeRegister)
	return nil
}

func (us *userService) DeleteAccount(ctx context.Context, userID uint64, password string) error {
	if userID == 0 {
		return errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}
	if strings.TrimSpace(password) == "" {
		return errors.NewCode(errcode.ErrValidation, "密码不能为空")
	}
	if _, err := us.tokenVersionStore(); err != nil {
		return err
	}

	users, err := us.usersData()
	if err != nil {
		return err
	}
	user, err := users.GetAuth(ctx, userID)
	if err != nil {
		return err
	}
	if err = users.CheckPassWord(ctx, password, user.PasswordHash); err != nil {
		return err
	}
	if err = users.Delete(ctx, userID); err != nil {
		return err
	}
	if err = us.bumpTokenVersion(ctx, userID); err != nil {
		return errors.NewCode(bizcode.ErrConnectGRPC, "注销账号失败")
	}
	if sessions, ok := us.sessionData(); ok {
		_ = sessions.RevokeAllSessions(ctx, userID)
	}
	return nil
}

func (u *userService) Update(ctx context.Context, userDTO *UserDTO) error {
	if userDTO == nil || userDTO.ID == 0 {
		return errors.NewCode(errcode.ErrValidation, "用户信息不能为空")
	}

	users, err := u.usersData()
	if err != nil {
		return err
	}
	return users.Update(ctx, &userDTO.User)
}

func (us *userService) Get(ctx context.Context, userID uint64) (*UserDTO, error) {
	if userID == 0 {
		return nil, errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}

	users, err := us.usersData()
	if err != nil {
		return nil, err
	}
	userDO, err := users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserDTO{User: userDO}, nil
}

func (u *userService) GetByUsername(ctx context.Context, username string) (*UserDTO, error) {
	username = normalizeLoginIdentifier(username)
	if username == "" {
		return nil, errors.NewCode(bizcode.ErrUserNotFound, "用户不存在")
	}

	users, err := u.usersData()
	if err != nil {
		return nil, err
	}
	userDO, err := users.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return &UserDTO{User: userDO}, nil
}

func (us *userService) usersData() (data.UserData, error) {
	if us == nil || us.data == nil {
		return nil, errors.NewCode(bizcode.ErrConnectGRPC, "user data client is not initialized")
	}
	users := us.data.Users()
	if users == nil {
		return nil, errors.NewCode(bizcode.ErrConnectGRPC, "user data client is not initialized")
	}
	return users, nil
}

func (us *userService) tokenVersionStore() (tokenversion.Store, error) {
	if us == nil || us.tokenVersions == nil {
		return nil, errors.NewCode(bizcode.ErrConnectGRPC, "token version store is not initialized")
	}
	return us.tokenVersions, nil
}

func isContextError(err error) bool {
	return stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded)
}

var _ UserSrv = &userService{}
