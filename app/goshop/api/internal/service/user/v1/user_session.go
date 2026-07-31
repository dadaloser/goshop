package v1

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"goshop/gmicro/errcode"
	"strings"
	"time"

	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/authz"
	"goshop/app/pkg/bizcode"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type sessionData interface {
	RecordLogin(ctx context.Context, userID uint64, at time.Time) error
	CreateSession(ctx context.Context, userID uint64, deviceID, deviceName, refreshToken string, expiresAt time.Time) (data.Session, error)
	RefreshSession(ctx context.Context, sessionID, currentToken, nextToken string, expiresAt time.Time) (data.Session, error)
	RevokeSession(ctx context.Context, userID uint64, sessionID string) error
	RevokeAllSessions(ctx context.Context, userID uint64) error
	ValidateSession(ctx context.Context, userID uint64, sessionID string) (bool, error)
	ListUserSessions(ctx context.Context, userID uint64, page, pageSize int) (data.SessionList, error)
}

func (us *userService) ListDevices(ctx context.Context, userID uint64, page, pageSize int) (data.SessionList, error) {
	sessions, ok := us.sessionData()
	if !ok {
		return data.SessionList{}, errors.NewSpec(bizcode.ConnectGRPCSpec, "session store is not configured")
	}
	return sessions.ListUserSessions(ctx, userID, page, pageSize)
}

func (us *userService) LogoutDevice(ctx context.Context, userID uint64, sessionID string) error {
	sessions, ok := us.sessionData()
	if !ok {
		return errors.NewSpec(bizcode.ConnectGRPCSpec, "session store is not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.NewSpec(bizcode.UserAccountInactiveSpec, "session id is required")
	}
	return sessions.RevokeSession(ctx, userID, sessionID)
}

func (us *userService) ListDeviceBlacklist(ctx context.Context, page, pageSize int) (data.DeviceBlacklistList, error) {
	store, ok := us.deviceBlacklistData()
	if !ok {
		return data.DeviceBlacklistList{}, errors.NewSpec(bizcode.ConnectGRPCSpec, "device blacklist store is not configured")
	}
	return store.ListDeviceBlacklist(ctx, page, pageSize)
}
func (us *userService) AddDeviceBlacklist(ctx context.Context, userID uint64, deviceID string) error {
	store, ok := us.deviceBlacklistData()
	if !ok {
		return errors.NewSpec(bizcode.ConnectGRPCSpec, "device blacklist store is not configured")
	}
	return store.AddDeviceBlacklist(ctx, userID, deviceID)
}
func (us *userService) DeleteDeviceBlacklist(ctx context.Context, userID uint64, deviceID string) error {
	store, ok := us.deviceBlacklistData()
	if !ok {
		return errors.NewSpec(bizcode.ConnectGRPCSpec, "device blacklist store is not configured")
	}
	return store.DeleteDeviceBlacklist(ctx, userID, deviceID)
}
func (us *userService) deviceBlacklistData() (data.DeviceBlacklistData, bool) {
	if us == nil || us.data == nil {
		return nil, false
	}
	store, ok := us.data.Users().(data.DeviceBlacklistData)
	return store, ok
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
		return nil, errors.NewSpec(errcode.TokenInvalidSpec, "refresh token invalid")
	}
	nextToken := secureToken()
	now := time.Now()
	refreshExpiresAt := now.Add(us.jwtOpts.MaxRefresh)
	session, err := sessions.RefreshSession(ctx, sessionID, refreshToken, nextToken, refreshExpiresAt)
	if err != nil {
		return nil, errors.NewSpec(errcode.TokenInvalidSpec, "refresh token invalid")
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

func (us *userService) createToken(ctx context.Context, user data.UserAuth) (string, int64, string, int64, string, error) {
	if us == nil || us.jwtOpts == nil || strings.TrimSpace(us.jwtOpts.Key) == "" || us.jwtOpts.Timeout <= 0 {
		return "", 0, "", 0, "", errors.NewSpec(bizcode.ConnectGRPCSpec, "jwt options are not initialized")
	}
	status := authz.NormalizeAccountStatus(user.Status)
	if status != authz.AccountStatusActive {
		return "", 0, "", 0, "", errors.NewSpec(bizcode.UserAccountInactiveSpec, "user account is inactive")
	}

	now := time.Now()
	var refreshToken, sessionID string
	var refreshExpiresAt time.Time
	if sessions, ok := us.sessionData(); ok {
		refreshToken = secureToken()
		if refreshToken == "" {
			return "", 0, "", 0, "", errors.NewCode(errcode.ErrUnknown, "create refresh token failed")
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
	deviceID := strings.TrimSpace(headers.GetHeader("X-Device-Instance-ID"))
	deviceName := strings.TrimSpace(headers.GetHeader("X-Device-Name"))
	if deviceID == "" {
		deviceID = uuid.NewString()
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
		return "", 0, errors.NewSpec(bizcode.UserAccountInactiveSpec, "user account is inactive")
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
			NotBefore: jwt.NewNumericDate(now),
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
		return errors.NewSpec(bizcode.UserNotFoundSpec, "user id is required")
	}

	if err := us.bumpTokenVersion(ctx, userID); err != nil {
		return errors.NewSpec(bizcode.ConnectGRPCSpec, "bump token version")
	}
	if sessions, ok := us.sessionData(); ok {
		if err := sessions.RevokeAllSessions(ctx, userID); err != nil {
			return errors.NewSpec(bizcode.ConnectGRPCSpec, "revoke all sessions")
		}
	}
	return nil
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
