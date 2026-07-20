package v1

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	"goshop/app/goshop/api/internal/loginattempt"
	"goshop/app/goshop/api/internal/smsattempt"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	"goshop/app/goshop/api/internal/data"
	"goshop/app/goshop/api/internal/smscode"
	"goshop/app/pkg/authsession/tokenversion"
	"goshop/app/pkg/authz"
	"goshop/app/pkg/options"
	code2 "goshop/gmicro/code"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/storage"

	"github.com/golang-jwt/jwt/v5"
)

type UserDTO struct {
	data.User

	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

type UserSrv interface {
	PasswordLogin(ctx context.Context, username, password string) (*UserDTO, error)
	SmsLogin(ctx context.Context, mobile, smsCode string) (*UserDTO, error)
	Register(ctx context.Context, mobile, email, username, password, nickName, code string) (*UserDTO, error)
	Update(ctx context.Context, userDTO *UserDTO) error
	Get(ctx context.Context, userID uint64) (*UserDTO, error)
	GetByUsername(ctx context.Context, username string) (*UserDTO, error)
	LogoutAll(ctx context.Context, userID uint64) error
	DeleteAccount(ctx context.Context, userID uint64, password string) error
}

type userService struct {
	//ud data.UserData
	data data.DataFactory

	jwtOpts *options.JwtOptions

	codeStore smscode.Store

	loginAttempts loginattempt.Store

	smsAttempts smsattempt.Store

	tokenVersions tokenversion.Store
}

func NewUserService(data data.DataFactory, jwtOpts *options.JwtOptions, codeStore smscode.Store, loginAttempts loginattempt.Store, smsAttempts smsattempt.Store, tokenVersions tokenversion.Store) UserSrv {
	return &userService{data: data, jwtOpts: jwtOpts, codeStore: codeStore, loginAttempts: loginAttempts, smsAttempts: smsAttempts, tokenVersions: tokenVersions}
}

func (us *userService) PasswordLogin(ctx context.Context, username, password string) (*UserDTO, error) {
	username = normalizeLoginIdentifier(username)
	users, err := us.usersData()
	if err != nil {
		return nil, err
	}
	if err := us.ensurePasswordLoginAllowed(ctx, username); err != nil {
		return nil, err
	}

	user, err := users.GetAuthByUsername(ctx, username)
	if err != nil {
		if errors.IsCode(err, code.ErrUserNotFound) {
			if lockedErr := us.recordPasswordLoginFailure(ctx, username); lockedErr != nil {
				return nil, lockedErr
			}
			return nil, errors.WithCode(code.ErrUserPasswordIncorrect, "手机号或密码错误")
		}
		return nil, err
	}

	//检查密码是否正确
	err = users.CheckPassWord(ctx, password, user.PasswordHash)
	if err != nil {
		if errors.IsCode(err, code.ErrUserPasswordIncorrect) {
			if lockedErr := us.recordPasswordLoginFailure(ctx, username); lockedErr != nil {
				return nil, lockedErr
			}
			return nil, errors.WithCode(code.ErrUserPasswordIncorrect, "手机号或密码错误")
		}
		return nil, err
	}

	us.resetPasswordLoginFailures(ctx, username)

	token, expiresAt, err := us.createToken(ctx, user)
	if err != nil {
		return nil, err
	}

	return &UserDTO{
		User:      user.User,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (us *userService) SmsLogin(ctx context.Context, mobile, smsCode string) (*UserDTO, error) {
	if err := us.ensureSmsCodeAllowed(ctx, mobile, smscode.TypeLogin); err != nil {
		return nil, err
	}
	if us == nil || us.codeStore == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "sms code store is not initialized")
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
		return nil, errors.WithCode(code.ErrCodeNotExist, "验证码不存在")
	}
	if value != smsCode {
		if lockedErr := us.recordSmsCodeFailure(ctx, mobile, smscode.TypeLogin); lockedErr != nil {
			return nil, lockedErr
		}
		return nil, errors.WithCode(code.ErrCodeInCorrect, "验证码错误")
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

	if ok := us.codeStore.Delete(ctx, key); !ok {
		log.Warn("delete sms login code failed")
	}

	token, expiresAt, err := us.createToken(ctx, user)
	if err != nil {
		return nil, err
	}
	return &UserDTO{
		User:      user.User,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (us *userService) Register(ctx context.Context, mobile, email, username, password, nickName, codes string) (*UserDTO, error) {
	if err := us.ensureSmsCodeAllowed(ctx, mobile, smscode.TypeRegister); err != nil {
		return nil, err
	}
	if us == nil || us.codeStore == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "sms code store is not initialized")
	}

	key := smscode.RegisterKey(mobile)
	value, err := us.codeStore.Get(ctx, key)
	if err != nil {
		if isContextError(err) {
			return nil, err
		}
		if !stderrors.Is(err, storage.ErrKeyNotFound) {
			return nil, err
		}
		if lockedErr := us.recordSmsCodeFailure(ctx, mobile, smscode.TypeRegister); lockedErr != nil {
			return nil, lockedErr
		}
		return nil, errors.WithCode(code.ErrCodeNotExist, "验证码不存在")
	}

	if value != codes {
		if lockedErr := us.recordSmsCodeFailure(ctx, mobile, smscode.TypeRegister); lockedErr != nil {
			return nil, lockedErr
		}
		return nil, errors.WithCode(code.ErrCodeInCorrect, "验证码错误")
	}

	us.resetSmsCodeFailures(ctx, mobile, smscode.TypeRegister)

	var user = &data.UserCreate{
		Username: username,
		Mobile:   mobile,
		Email:    email,
		NickName: nickName,
		PassWord: password,
	}
	users, err := us.usersData()
	if err != nil {
		return nil, err
	}
	createdUser, err := users.Create(ctx, user)
	if err != nil {
		log.Errorf("user register failed: %v", err)
		return nil, err
	}

	if ok := us.codeStore.Delete(ctx, key); !ok {
		log.Warn("delete sms code failed")
	}

	token, expiresAt, err := us.createToken(ctx, data.UserAuth{User: createdUser, PasswordHash: ""})
	if err != nil {
		return nil, err
	}

	return &UserDTO{
		User:      createdUser,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (us *userService) createToken(ctx context.Context, user data.UserAuth) (string, int64, error) {
	if us == nil || us.jwtOpts == nil || strings.TrimSpace(us.jwtOpts.Key) == "" || us.jwtOpts.Timeout <= 0 {
		return "", 0, errors.WithCode(code.ErrConnectGRPC, "jwt options are not initialized")
	}
	status := authz.NormalizeAccountStatus(user.Status)
	if status != authz.AccountStatusActive {
		return "", 0, errors.WithCode(code.ErrUserAccountInactive, "用户账户不可用")
	}

	now := time.Now()
	j := middlewares.NewJWT(us.jwtOpts.Key)
	claims := middlewares.CustomClaims{
		ID:            uint(user.ID),
		NickName:      user.NickName,
		AuthorityId:   uint(user.LegacyRole),
		PrincipalType: string(authz.PrincipalCustomer),
		AccountStatus: string(status),
		Scope:         authz.CustomerScopes(),
		TokenVersion:  us.currentTokenVersion(ctx, user.ID),
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(now), //签名的生效时间
			ExpiresAt: jwt.NewNumericDate(now.Add(us.jwtOpts.Timeout)),
			Issuer:    us.jwtOpts.Realm,
		},
	}
	token, err := j.CreateToken(claims)
	if err != nil {
		return "", 0, err
	}
	return token, now.Local().Add(us.jwtOpts.Timeout).Unix(), nil
}

func (us *userService) LogoutAll(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	if err := us.bumpTokenVersion(ctx, userID); err != nil {
		return errors.WithCode(code.ErrConnectGRPC, "退出登录失败")
	}
	return nil
}

func (us *userService) DeleteAccount(ctx context.Context, userID uint64, password string) error {
	if userID == 0 {
		return errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}
	if strings.TrimSpace(password) == "" {
		return errors.WithCode(code2.ErrValidation, "密码不能为空")
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
		return errors.WithCode(code.ErrConnectGRPC, "注销账号失败")
	}
	return nil
}

func (u *userService) Update(ctx context.Context, userDTO *UserDTO) error {
	if userDTO == nil || userDTO.ID == 0 {
		return errors.WithCode(code2.ErrValidation, "用户信息不能为空")
	}

	users, err := u.usersData()
	if err != nil {
		return err
	}
	return users.Update(ctx, &userDTO.User)
}

func (us *userService) Get(ctx context.Context, userID uint64) (*UserDTO, error) {
	if userID == 0 {
		return nil, errors.WithCode(code.ErrUserNotFound, "用户不存在")
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
		return nil, errors.WithCode(code.ErrUserNotFound, "用户不存在")
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
		return nil, errors.WithCode(code.ErrConnectGRPC, "user data client is not initialized")
	}
	users := us.data.Users()
	if users == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "user data client is not initialized")
	}
	return users, nil
}

func (us *userService) tokenVersionStore() (tokenversion.Store, error) {
	if us == nil || us.tokenVersions == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "token version store is not initialized")
	}
	return us.tokenVersions, nil
}

func (us *userService) currentTokenVersion(ctx context.Context, userID uint64) uint64 {
	if us == nil || us.tokenVersions == nil || userID == 0 {
		return 0
	}

	version, err := us.tokenVersions.CurrentVersion(ctx, userID)
	if err != nil {
		log.Errorf("load token version failed: userID=%d error=%v", userID, err)
		return 0
	}
	return version
}

func (us *userService) bumpTokenVersion(ctx context.Context, userID uint64) error {
	store, err := us.tokenVersionStore()
	if err != nil {
		return err
	}
	if _, err = store.Bump(ctx, userID); err != nil {
		log.Errorf("bump token version failed: userID=%d error=%v", userID, err)
		return err
	}
	return nil
}

func isContextError(err error) bool {
	return stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded)
}

func (us *userService) ensurePasswordLoginAllowed(ctx context.Context, username string) error {
	if us == nil || us.loginAttempts == nil {
		return nil
	}

	locked, err := us.loginAttempts.IsLocked(ctx, username)
	if err != nil {
		log.Errorf("check password login attempts failed: %v", err)
		return errors.WithCode(code.ErrUserLoginLocked, "登录暂时不可用，请稍后重试")
	}
	if locked {
		return errors.WithCode(code.ErrUserLoginLocked, "登录失败次数过多，请稍后重试")
	}
	return nil
}

func (us *userService) recordPasswordLoginFailure(ctx context.Context, username string) error {
	if us == nil || us.loginAttempts == nil {
		return nil
	}

	locked, err := us.loginAttempts.RecordFailure(ctx, username)
	if err != nil {
		log.Errorf("record password login failure failed: %v", err)
		return errors.WithCode(code.ErrUserLoginLocked, "登录暂时不可用，请稍后重试")
	}
	if locked {
		return errors.WithCode(code.ErrUserLoginLocked, "登录失败次数过多，请稍后重试")
	}
	return nil
}

func (us *userService) resetPasswordLoginFailures(ctx context.Context, username string) {
	if us == nil || us.loginAttempts == nil {
		return
	}

	if err := us.loginAttempts.Reset(ctx, username); err != nil {
		log.Warnf("reset password login failures failed: %v", err)
	}
}

func (us *userService) ensureSmsCodeAllowed(ctx context.Context, mobile string, codeType uint) error {
	if us == nil || us.smsAttempts == nil {
		return nil
	}

	locked, err := us.smsAttempts.IsLocked(ctx, mobile, codeType)
	if err != nil {
		log.Errorf("check sms verification attempts failed: %v", err)
		return errors.WithCode(code.ErrSmsVerifyLocked, "验证码验证暂时不可用，请稍后重试")
	}
	if locked {
		return errors.WithCode(code.ErrSmsVerifyLocked, "验证码错误次数过多，请稍后重试")
	}
	return nil
}

func (us *userService) recordSmsCodeFailure(ctx context.Context, mobile string, codeType uint) error {
	if us == nil || us.smsAttempts == nil {
		return nil
	}

	locked, err := us.smsAttempts.RecordFailure(ctx, mobile, codeType)
	if err != nil {
		log.Errorf("record sms verification failure failed: %v", err)
		return errors.WithCode(code.ErrSmsVerifyLocked, "验证码验证暂时不可用，请稍后重试")
	}
	if locked {
		return errors.WithCode(code.ErrSmsVerifyLocked, "验证码错误次数过多，请稍后重试")
	}
	return nil
}

func (us *userService) resetSmsCodeFailures(ctx context.Context, mobile string, codeType uint) {
	if us == nil || us.smsAttempts == nil {
		return
	}

	if err := us.smsAttempts.Reset(ctx, mobile, codeType); err != nil {
		log.Warnf("reset sms verification failures failed: %v", err)
	}
}

func normalizeLoginIdentifier(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

var _ UserSrv = &userService{}
