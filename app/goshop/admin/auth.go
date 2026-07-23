package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	upbv1 "goshop/api/user/v1"
	"goshop/app/goshop/admin/config"
	"goshop/app/pkg/authsession/tokenrevocation"
	"goshop/app/pkg/authsession/tokenversion"
	"goshop/app/pkg/authz"
	"goshop/app/pkg/options"
	"goshop/gmicro/code"
	"goshop/gmicro/core/metric"
	"goshop/gmicro/server/restserver/middlewares"
	gauth "goshop/gmicro/server/restserver/middlewares/auth"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var breakGlassEvents = metric.NewCounterVec(&metric.CounterVecOpts{Namespace: "goshop", Subsystem: "admin", Name: "break_glass_events_total", Help: "Break-glass issuance and audit outcomes", Labels: []string{"outcome"}})

type staffAuthHandler struct {
	users         upbv1.UserClient
	jwtOpts       *options.JwtOptions
	adminAuth     *config.AdminAuthOptions
	revokedTokens tokenrevocation.Store
	tokenVersions tokenversion.Store
}

type staffLoginRequest struct {
	Username string `json:"username" binding:"required,min=1,max=100"`
	Password string `json:"password" binding:"required,min=1,max=72"`
}

func newStaffAuthHandler(
	users upbv1.UserClient,
	jwtOpts *options.JwtOptions,
	adminAuth *config.AdminAuthOptions,
	revokedTokens tokenrevocation.Store,
	tokenVersions tokenversion.Store,
) *staffAuthHandler {
	return &staffAuthHandler{
		users:         users,
		jwtOpts:       jwtOpts,
		adminAuth:     adminAuth,
		revokedTokens: revokedTokens,
		tokenVersions: tokenVersions,
	}
}

func (h *staffAuthHandler) Login(ctx *gin.Context) {
	if h == nil || h.users == nil || h.jwtOpts == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "staff auth is not initialized",
		})
		return
	}

	var request staffLoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code": http.StatusBadRequest,
			"msg":  "invalid request",
		})
		return
	}

	identifier := strings.ToLower(strings.TrimSpace(request.Username))
	authUser, err := h.users.GetUserAuthByMobile(ctx.Request.Context(), &upbv1.MobileRequest{Mobile: identifier})
	if err != nil {
		if status.Code(err) == codes.NotFound || status.Code(err) == codes.InvalidArgument {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "账号或密码错误",
			})
			return
		}
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "staff auth backend unavailable",
		})
		return
	}
	if authUser == nil || authUser.GetUser() == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": http.StatusUnauthorized,
			"msg":  "账号或密码错误",
		})
		return
	}
	if authz.NormalizeAccountStatus(authUser.GetUser().GetStatus()) != authz.AccountStatusActive {
		ctx.JSON(http.StatusForbidden, gin.H{
			"code": http.StatusForbidden,
			"msg":  "staff account is not active",
		})
		return
	}
	if len(authUser.GetStaffRoles()) == 0 || len(authUser.GetPermissions()) == 0 {
		ctx.JSON(http.StatusForbidden, gin.H{
			"code": http.StatusForbidden,
			"msg":  "staff role is required",
		})
		return
	}

	check, err := h.users.CheckPassWord(ctx.Request.Context(), &upbv1.PasswordCheckInfo{
		Password:          request.Password,
		EncryptedPassword: authUser.GetPasswordHash(),
	})
	if err != nil || check == nil || !check.GetSuccess() {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": http.StatusUnauthorized,
			"msg":  "账号或密码错误",
		})
		return
	}

	token, expiresAt, err := h.createToken(ctx.Request.Context(), authUser)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"msg":  "create staff token failed",
		})
		return
	}
	if err = h.createAdminAuditLog(ctx.Request.Context(), &upbv1.AdminAuditLog{
		TargetUserId:       authUser.GetUser().GetId(),
		ActorUserId:        authUser.GetUser().GetId(),
		ActorPrincipalType: string(authz.PrincipalStaff),
		Action:             "staff_login_succeeded",
		Detail:             fmt.Sprintf("roles:%s", strings.Join(authUser.GetStaffRoles(), ",")),
	}); err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "staff login audit failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"token":       token,
		"expires_at":  expiresAt,
		"user":        authUser.GetUser(),
		"roles":       authUser.GetStaffRoles(),
		"permissions": authUser.GetPermissions(),
	})
}

func (h *staffAuthHandler) Logout(ctx *gin.Context) {
	if h == nil || h.revokedTokens == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "staff revocation store is not initialized",
		})
		return
	}

	token, err := gauth.GetToken(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": http.StatusUnauthorized,
			"msg":  "token not found",
		})
		return
	}

	expiresAt, err := jwtExpiresAt(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": http.StatusUnauthorized,
			"msg":  "token exp invalid",
		})
		return
	}

	if err = h.revokedTokens.Revoke(ctx.Request.Context(), token, expiresAt); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"msg":  "staff logout failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *staffAuthHandler) LogoutAll(ctx *gin.Context) {
	if h == nil || h.tokenVersions == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "staff token version store is not initialized",
		})
		return
	}

	userID, err := userIDFromClaims(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": http.StatusUnauthorized,
			"msg":  "token user id invalid",
		})
		return
	}

	if _, err = h.tokenVersions.Bump(ctx.Request.Context(), userID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"msg":  "staff logout_all failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *staffAuthHandler) Me(ctx *gin.Context) {
	if h == nil || h.users == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "staff auth backend is not initialized",
		})
		return
	}

	userID, err := userIDFromClaims(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code": http.StatusUnauthorized,
			"msg":  "token user id invalid",
		})
		return
	}

	authUser, err := h.users.GetUserAuthById(ctx.Request.Context(), &upbv1.IdRequest{Id: int32(userID)})
	if err != nil || authUser == nil || authUser.GetUser() == nil {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "staff profile lookup failed",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user":           authUser.GetUser(),
		"roles":          authUser.GetStaffRoles(),
		"permissions":    authUser.GetPermissions(),
		"principal_type": authz.PrincipalStaff,
	})
}

func (h *staffAuthHandler) BootstrapSession(ctx *gin.Context) {
	if h == nil || h.jwtOpts == nil || h.adminAuth == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "bootstrap auth is not initialized",
		})
		return
	}

	timeout := h.adminAuth.EffectiveBreakGlassTTL()
	if h.jwtOpts.Timeout > 0 && h.jwtOpts.Timeout < timeout {
		timeout = h.jwtOpts.Timeout
	}

	now := time.Now()
	correlationID := uuid.NewString()
	keyID := h.adminAuth.EffectiveBreakGlassKeyID()
	token, err := middlewares.NewJWT(h.jwtOpts.Key).CreateToken(middlewares.CustomClaims{
		PrincipalType: string(authz.PrincipalAdminBootstrap),
		AccountStatus: string(authz.AccountStatusActive),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        correlationID,
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(timeout)),
			Issuer:    h.jwtOpts.Realm,
		},
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"msg":  "create bootstrap session failed",
		})
		return
	}
	if err = h.createAdminAuditLog(ctx.Request.Context(), &upbv1.AdminAuditLog{
		ActorPrincipalType: string(authz.PrincipalAdminBootstrap),
		Action:             "break_glass_session_issued",
		Detail:             fmt.Sprintf("correlation_id:%s key_id:%s grants:none", correlationID, keyID),
		CorrelationId:      correlationID,
		RequestId:          requestID(ctx),
		TargetType:         "break_glass_session",
		TargetId:           correlationID,
		Domain:             string(authz.BusinessDomainPlatform),
	}); err != nil {
		breakGlassEvents.Inc("audit_failed")
		ctx.JSON(http.StatusBadGateway, gin.H{
			"code": http.StatusBadGateway,
			"msg":  "create bootstrap session audit failed",
		})
		return
	}
	breakGlassEvents.Inc("issued")
	log.Warnf("SECURITY_ALERT break-glass session issued correlation_id=%s key_id=%s expires_at=%d", correlationID, keyID, now.Add(timeout).Unix())

	ctx.JSON(http.StatusOK, gin.H{
		"token":          token,
		"expires_at":     now.Add(timeout).Unix(),
		"principal_type": authz.PrincipalAdminBootstrap,
		"correlation_id": correlationID,
		"key_id":         keyID,
	})
}

func (h *staffAuthHandler) createToken(ctx context.Context, authUser *upbv1.UserAuthResponse) (string, int64, error) {
	tokenVersion, err := h.currentTokenVersion(ctx, authUser.GetUser().GetId())
	if err != nil {
		return "", 0, err
	}
	now := time.Now()
	token, err := middlewares.NewJWT(h.jwtOpts.Key).CreateToken(middlewares.CustomClaims{
		ID:              uint(authUser.GetUser().GetId()),
		NickName:        authUser.GetUser().GetNickName(),
		AuthorityId:     uint(authUser.GetLegacyRole()),
		Roles:           append([]string(nil), authUser.GetStaffRoles()...),
		PrincipalType:   string(authz.PrincipalStaff),
		AccountStatus:   authUser.GetUser().GetStatus(),
		Scope:           append([]string(nil), authUser.GetPermissions()...),
		TokenVersion:    tokenVersion,
		ResourceDomains: effectiveResourceDomains(authUser),
		ResourceStores:  append([]string(nil), authUser.GetResourceStores()...),
		ResourceTeams:   append([]string(nil), authUser.GetResourceTeams()...),
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(h.jwtOpts.Timeout)),
			Issuer:    h.jwtOpts.Realm,
		},
	})
	if err != nil {
		return "", 0, err
	}
	return token, now.Add(h.jwtOpts.Timeout).Unix(), nil
}

func roleDomains(roles []string) []string {
	seen := map[string]struct{}{}
	for _, definition := range authz.BuiltinRoleDefinitions() {
		for _, role := range roles {
			if strings.EqualFold(role, string(definition.Name)) {
				for _, domain := range definition.Domains {
					seen[string(domain)] = struct{}{}
				}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for domain := range seen {
		result = append(result, domain)
	}
	sort.Strings(result)
	return result
}

func effectiveResourceDomains(user *upbv1.UserAuthResponse) []string {
	if len(user.GetResourceDomains()) > 0 {
		return append([]string(nil), user.GetResourceDomains()...)
	}
	return roleDomains(user.GetStaffRoles())
}

func (h *staffAuthHandler) currentTokenVersion(ctx context.Context, userID int32) (uint64, error) {
	if h == nil || h.tokenVersions == nil || userID <= 0 {
		return 0, nil
	}
	return h.tokenVersions.CurrentVersion(ctx, uint64(userID))
}

func (h *staffAuthHandler) createAdminAuditLog(ctx context.Context, logEntry *upbv1.AdminAuditLog) error {
	if h == nil || h.users == nil || logEntry == nil {
		return nil
	}
	_, err := h.users.CreateAdminAuditLog(ctx, &upbv1.CreateAdminAuditLogRequest{Log: logEntry})
	return err
}

func newStaffJWTAuth(
	opts *options.JwtOptions,
	revokedTokens tokenrevocation.Store,
	tokenVersions tokenversion.Store,
	users upbv1.UserClient,
) (middlewares.AuthStrategy, error) {
	if opts == nil {
		return nil, status.Error(codes.InvalidArgument, "jwt options are required")
	}
	return gauth.NewJWTStrategy([]byte(opts.Key), opts.Realm, middlewares.KeyUserID, func(_ interface{}, c *gin.Context) bool {
		claims := gauth.ExtractClaims(c)
		principalType, _ := claims["principal_type"].(string)
		statusValue, _ := claims["status"].(string)
		if principalType != string(authz.PrincipalStaff) ||
			authz.NormalizeAccountStatus(statusValue) != authz.AccountStatusActive {
			return false
		}

		token, err := gauth.GetToken(c)
		if err != nil {
			return false
		}
		if revokedTokens != nil {
			revoked, err := revokedTokens.IsRevoked(c.Request.Context(), token)
			if err != nil || revoked {
				return false
			}
		}

		userID, err := userIDFromClaims(c)
		if err != nil {
			return false
		}
		if users != nil {
			authUser, err := users.GetUserAuthById(c.Request.Context(), &upbv1.IdRequest{Id: int32(userID)})
			if err != nil || authUser == nil || authUser.GetUser() == nil {
				return false
			}
			if authz.NormalizeAccountStatus(authUser.GetUser().GetStatus()) != authz.AccountStatusActive {
				return false
			}
		}
		if tokenVersions != nil {
			currentVersion, err := tokenVersions.CurrentVersion(c.Request.Context(), userID)
			if err != nil || currentVersion != tokenVersionFromClaims(claims) {
				return false
			}
		}
		return true
	}), nil
}

func jwtExpiresAt(ctx *gin.Context) (time.Time, error) {
	exp, ok := gauth.ExtractClaims(ctx)["exp"]
	if !ok {
		return time.Time{}, errors.WithCode(code.ErrTokenInvalid, "token missing exp")
	}

	var unix int64
	switch value := exp.(type) {
	case float64:
		unix = int64(value)
	case json.Number:
		v, err := value.Int64()
		if err != nil {
			return time.Time{}, errors.WithCode(code.ErrTokenInvalid, "token exp invalid")
		}
		unix = v
	case int64:
		unix = value
	case int:
		unix = int64(value)
	default:
		return time.Time{}, errors.WithCode(code.ErrTokenInvalid, "token exp invalid")
	}
	if unix <= 0 {
		return time.Time{}, errors.WithCode(code.ErrTokenInvalid, "token exp invalid")
	}
	return time.Unix(unix, 0), nil
}

func userIDFromClaims(ctx *gin.Context) (uint64, error) {
	claims := gauth.ExtractClaims(ctx)
	switch value := claims["user_id"].(type) {
	case float64:
		if value > 0 {
			return uint64(value), nil
		}
	case uint64:
		if value > 0 {
			return value, nil
		}
	case uint:
		if value > 0 {
			return uint64(value), nil
		}
	case int:
		if value > 0 {
			return uint64(value), nil
		}
	case int64:
		if value > 0 {
			return uint64(value), nil
		}
	}
	return 0, errors.WithCode(code.ErrTokenInvalid, "token user id invalid")
}

func tokenVersionFromClaims(claims map[string]any) uint64 {
	switch value := claims["token_version"].(type) {
	case float64:
		if value > 0 {
			return uint64(value)
		}
	case uint64:
		return value
	case int64:
		if value > 0 {
			return uint64(value)
		}
	case int:
		if value > 0 {
			return uint64(value)
		}
	}
	return 0
}
