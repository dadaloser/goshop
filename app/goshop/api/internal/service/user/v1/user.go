package v1

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	stderrors "errors"
	"strings"
	"time"

	"goshop/app/goshop/api/internal/emailcode"
	"goshop/app/goshop/api/internal/loginattempt"
	"goshop/app/goshop/api/internal/smsattempt"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	opb "goshop/api/order/v1"
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

	Token            string `json:"token"`
	ExpiresAt        int64  `json:"expires_at"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	RefreshExpiresAt int64  `json:"refresh_expires_at,omitempty"`
	SessionID        string `json:"session_id,omitempty"`
}

type sessionData interface {
	RecordLogin(ctx context.Context, userID uint64, at time.Time) error
	CreateSession(ctx context.Context, userID uint64, deviceID, deviceName, refreshToken string, expiresAt time.Time) (data.Session, error)
	RefreshSession(ctx context.Context, sessionID, currentToken, nextToken string, expiresAt time.Time) (data.Session, error)
	RevokeSession(ctx context.Context, userID uint64, sessionID string) error
	RevokeAllSessions(ctx context.Context, userID uint64) error
	ValidateSession(ctx context.Context, userID uint64, sessionID string) (bool, error)
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
	DeleteAccount(ctx context.Context, userID uint64, password string) error
}

func (us *userService) Logout(ctx context.Context, userID uint64, sessionID string) error {
	if sessions, ok := us.sessionData(); ok && sessionID != "" {
		return sessions.RevokeSession(ctx, userID, sessionID)
	}
	return nil
}

func (us *userService) Refresh(ctx context.Context, sessionID, refreshToken string) (*UserDTO, error) {
	sessions, ok := us.sessionData()
	if !ok || strings.TrimSpace(sessionID) == "" || strings.TrimSpace(refreshToken) == "" {
		return nil, errors.WithCode(code2.ErrTokenInvalid, "refresh token invalid")
	}
	nextToken := secureToken()
	now := time.Now()
	refreshExpiresAt := now.Add(us.jwtOpts.MaxRefresh)
	session, err := sessions.RefreshSession(ctx, sessionID, refreshToken, nextToken, refreshExpiresAt)
	if err != nil {
		return nil, errors.WithCode(code2.ErrTokenInvalid, "refresh token invalid")
	}
	users, err := us.usersData()
	if err != nil {
		return nil, err
	}
	user, err := users.GetAuth(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	token, expiresAt, err := us.issueAccessToken(ctx, user, session.ID, now)
	if err != nil {
		return nil, err
	}
	return &UserDTO{User: user.User, Token: token, ExpiresAt: expiresAt, RefreshToken: nextToken, RefreshExpiresAt: refreshExpiresAt.Unix(), SessionID: session.ID}, nil
}

type userService struct {
	//ud data.UserData
	data data.DataFactory

	jwtOpts *options.JwtOptions

	codeStore smscode.Store

	loginAttempts loginattempt.Store

	smsAttempts smsattempt.Store

	tokenVersions tokenversion.Store
	emailCodes    emailcode.Store
}

func NewUserService(data data.DataFactory, jwtOpts *options.JwtOptions, codeStore smscode.Store, loginAttempts loginattempt.Store, smsAttempts smsattempt.Store, tokenVersions tokenversion.Store) UserSrv {
	return &userService{data: data, jwtOpts: jwtOpts, codeStore: codeStore, loginAttempts: loginAttempts, smsAttempts: smsAttempts, tokenVersions: tokenVersions, emailCodes: emailcode.NewRedisStore()}
}

func (us *userService) EmailLogin(ctx context.Context, email, verificationCode string) (*UserDTO, error) {
	email = normalizeLoginIdentifier(email)
	if us.emailCodes == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "email verification unavailable")
	}
	if err := us.emailCodes.Consume(ctx, email, "login", verificationCode); err != nil {
		return nil, errors.WithCode(code.ErrCodeInCorrect, "验证码错误或已失效")
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
		return nil, errors.WithCode(code.ErrUserAccountInactive, "邮箱尚未验证")
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
		return nil, errors.WithCode(code.ErrConnectGRPC, "email verification unavailable")
	}
	if err := us.emailCodes.Consume(ctx, email, "register", verificationCode); err != nil {
		return nil, errors.WithCode(code.ErrCodeInCorrect, "验证码错误或已失效")
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
		Username:       username,
		Mobile:         mobile,
		Email:          email,
		NickName:       nickName,
		PassWord:       password,
		MobileVerified: true,
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

	token, expiresAt, refreshToken, refreshExpiresAt, sessionID, err := us.createToken(ctx, data.UserAuth{User: createdUser, PasswordHash: ""})
	if err != nil {
		return nil, err
	}

	return &UserDTO{
		User:         createdUser,
		Token:        token,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken, RefreshExpiresAt: refreshExpiresAt, SessionID: sessionID,
	}, nil
}

func (us *userService) createToken(ctx context.Context, user data.UserAuth) (string, int64, string, int64, string, error) {
	if us == nil || us.jwtOpts == nil || strings.TrimSpace(us.jwtOpts.Key) == "" || us.jwtOpts.Timeout <= 0 {
		return "", 0, "", 0, "", errors.WithCode(code.ErrConnectGRPC, "jwt options are not initialized")
	}
	status := authz.NormalizeAccountStatus(user.Status)
	if status != authz.AccountStatusActive {
		return "", 0, "", 0, "", errors.WithCode(code.ErrUserAccountInactive, "用户账户不可用")
	}

	now := time.Now()
	var refreshToken, sessionID string
	var refreshExpiresAt time.Time
	if sessions, ok := us.sessionData(); ok {
		refreshToken = secureToken()
		if refreshToken == "" {
			return "", 0, "", 0, "", errors.WithCode(code2.ErrUnknown, "create refresh token failed")
		}
		refreshExpiresAt = now.Add(us.jwtOpts.MaxRefresh)
		deviceID, deviceName := loginDevice(ctx)
		session, err := sessions.CreateSession(ctx, user.ID, deviceID, deviceName, refreshToken, refreshExpiresAt)
		if err != nil {
			return "", 0, "", 0, "", err
		}
		sessionID = session.ID
		if err = sessions.RecordLogin(ctx, user.ID, now); err != nil {
			return "", 0, "", 0, "", err
		}
	}
	token, expiresAt, err := us.issueAccessToken(ctx, user, sessionID, now)
	if err != nil {
		return "", 0, "", 0, "", err
	}
	var refreshUnix int64
	if !refreshExpiresAt.IsZero() {
		refreshUnix = refreshExpiresAt.Unix()
	}
	return token, expiresAt, refreshToken, refreshUnix, sessionID, nil
}

func loginDevice(ctx context.Context) (string, string) {
	headers, ok := ctx.(interface{ GetHeader(string) string })
	if !ok {
		return "unknown", "unknown"
	}
	deviceID := strings.TrimSpace(headers.GetHeader("X-Device-ID"))
	deviceName := strings.TrimSpace(headers.GetHeader("X-Device-Name"))
	if deviceID == "" {
		deviceID = "unknown"
	}
	if deviceName == "" {
		deviceName = strings.TrimSpace(headers.GetHeader("User-Agent"))
	}
	if len(deviceID) > 128 {
		deviceID = deviceID[:128]
	}
	if len(deviceName) > 128 {
		deviceName = deviceName[:128]
	}
	return deviceID, deviceName
}

func (us *userService) issueAccessToken(ctx context.Context, user data.UserAuth, sessionID string, now time.Time) (string, int64, error) {
	status := authz.NormalizeAccountStatus(user.Status)
	if status != authz.AccountStatusActive {
		return "", 0, errors.WithCode(code.ErrUserAccountInactive, "用户账户不可用")
	}
	j := middlewares.NewJWT(us.jwtOpts.Key)
	claims := middlewares.CustomClaims{
		ID:            uint(user.ID),
		NickName:      user.NickName,
		AuthorityId:   uint(user.LegacyRole),
		PrincipalType: string(authz.PrincipalCustomer),
		AccountStatus: string(status),
		Scope:         authz.CustomerScopes(),
		TokenVersion:  us.currentTokenVersion(ctx, user.ID),
		SessionID:     sessionID,
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

func (us *userService) sessionData() (sessionData, bool) {
	users, err := us.usersData()
	if err != nil {
		return nil, false
	}
	sessions, ok := users.(sessionData)
	return sessions, ok
}

func secureToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func (us *userService) LogoutAll(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	if err := us.bumpTokenVersion(ctx, userID); err != nil {
		return errors.WithCode(code.ErrConnectGRPC, "退出登录失败")
	}
	if sessions, ok := us.sessionData(); ok {
		if err := sessions.RevokeAllSessions(ctx, userID); err != nil {
			return errors.WithCode(code.ErrConnectGRPC, "退出登录失败")
		}
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
	if err = us.ensureAccountDeletionAllowed(ctx, userID); err != nil {
		return err
	}
	if err = users.Delete(ctx, userID); err != nil {
		return err
	}
	if err = us.bumpTokenVersion(ctx, userID); err != nil {
		return errors.WithCode(code.ErrConnectGRPC, "注销账号失败")
	}
	if sessions, ok := us.sessionData(); ok {
		_ = sessions.RevokeAllSessions(ctx, userID)
	}
	return nil
}

func (us *userService) ensureAccountDeletionAllowed(ctx context.Context, userID uint64) error {
	if us == nil || us.data == nil {
		return errors.WithCode(code.ErrConnectGRPC, "无法检查未完成业务")
	}
	if us.data.Orders() == nil {
		return nil
	}
	const pageSize = 100
	for page := int32(1); ; page++ {
		resp, err := us.data.Orders().OrderList(ctx, &opb.OrderFilterRequest{UserId: int32(userID), Pages: page, PagePerNums: pageSize})
		if err != nil {
			return errors.WithCode(code.ErrConnectGRPC, "无法检查未完成业务")
		}
		for _, order := range resp.GetData() {
			status := strings.TrimSpace(order.GetStatus())
			if status != "TRADE_CLOSED" && status != "TRADE_FINISHED" {
				return errors.WithCode(code.ErrAccountDeletionBlocked, "存在未完成订单、退款或售后，暂不能注销")
			}
		}
		if len(resp.GetData()) < pageSize {
			return nil
		}
	}
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
