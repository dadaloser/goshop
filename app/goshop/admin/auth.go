package admin

import (
	"context"
	"crypto/rand"
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
	"goshop/gmicro/core/metric"
	"goshop/gmicro/server/restserver/middlewares"
	gauth "goshop/gmicro/server/restserver/middlewares/auth"
	"goshop/pkg/errcode"
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

type breakGlassApprovalRequest struct {
	RequesterUserID int32  `json:"requester_user_id" binding:"required"`
	Reason          string `json:"reason" binding:"required,min=3,max=255"`
	TTLSeconds      int64  `json:"ttl_seconds"`
}

type breakGlassApproveRequest struct {
	ApproverUserID int32 `json:"approver_user_id" binding:"required"`
}

type breakGlassSessionRequest struct {
	ApprovalID      string `json:"approval_id" binding:"required"`
	RequesterUserID int32  `json:"requester_user_id" binding:"required"`
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
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "staff auth is temporarily unavailable")
		return
	}

	var request staffLoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeValidationError(ctx, "invalid request")
		return
	}

	identifier := strings.ToLower(strings.TrimSpace(request.Username))
	authUser, err := h.users.GetUserAuthByMobile(ctx.Request.Context(), &upbv1.MobileRequest{Mobile: identifier})
	if err != nil {
		if status.Code(err) == codes.NotFound || status.Code(err) == codes.InvalidArgument {
			writePublicError(ctx, errcode.ErrTokenInvalid, errors.KindUnauthenticated, "账号或密码错误")
			return
		}
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "staff auth backend is temporarily unavailable")
		return
	}
	if authUser == nil || authUser.GetUser() == nil {
		writePublicError(ctx, errcode.ErrTokenInvalid, errors.KindUnauthenticated, "账号或密码错误")
		return
	}
	if authz.NormalizeAccountStatus(authUser.GetUser().GetStatus()) != authz.AccountStatusActive {
		writePublicError(ctx, errcode.ErrPermissionDenied, errors.KindPermissionDenied, "staff account is not active")
		return
	}
	if len(authUser.GetStaffRoles()) == 0 || len(authUser.GetPermissions()) == 0 {
		writePublicError(ctx, errcode.ErrPermissionDenied, errors.KindPermissionDenied, "staff role is required")
		return
	}

	check, err := h.users.CheckPassWord(ctx.Request.Context(), &upbv1.PasswordCheckInfo{
		Password:          request.Password,
		EncryptedPassword: authUser.GetPasswordHash(),
	})
	if err != nil || check == nil || !check.GetSuccess() {
		writePublicError(ctx, errcode.ErrTokenInvalid, errors.KindUnauthenticated, "账号或密码错误")
		return
	}

	sessionID, err := h.createStaffSession(ctx, authUser.GetUser().GetId())
	if err != nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "staff session service is temporarily unavailable")
		return
	}
	token, expiresAt, err := h.createToken(ctx.Request.Context(), authUser, sessionID)
	if err != nil {
		writePublicError(ctx, errcode.ErrUnknown, errors.KindInternal, "create staff token failed")
		return
	}
	if err = h.createAdminAuditLog(ctx.Request.Context(), &upbv1.AdminAuditLog{
		TargetUserId:       authUser.GetUser().GetId(),
		ActorUserId:        authUser.GetUser().GetId(),
		ActorPrincipalType: string(authz.PrincipalStaff),
		Action:             "staff_login_succeeded",
		Detail:             fmt.Sprintf("roles:%s session_id:%s", strings.Join(authUser.GetStaffRoles(), ","), sessionID),
	}); err != nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "staff audit service is temporarily unavailable")
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
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "staff revocation service is temporarily unavailable")
		return
	}

	token, err := gauth.GetToken(ctx)
	if err != nil {
		writePublicError(ctx, errcode.ErrTokenInvalid, errors.KindUnauthenticated, "token not found")
		return
	}

	expiresAt, err := jwtExpiresAt(ctx)
	if err != nil {
		writePublicError(ctx, errcode.ErrTokenInvalid, errors.KindUnauthenticated, "token exp invalid")
		return
	}

	if err = h.revokedTokens.Revoke(ctx.Request.Context(), token, expiresAt); err != nil {
		writePublicError(ctx, errcode.ErrUnknown, errors.KindInternal, "staff logout failed")
		return
	}
	if claims := gauth.ExtractClaims(ctx); claims != nil && h.users != nil {
		if sessionID, _ := claims["session_id"].(string); strings.TrimSpace(sessionID) != "" {
			_, _ = h.users.RevokeStaffSession(ctx.Request.Context(), &upbv1.RevokeStaffSessionRequest{SessionId: sessionID})
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": true})
}

func (h *staffAuthHandler) LogoutAll(ctx *gin.Context) {
	if h == nil || h.tokenVersions == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "员工令牌服务暂不可用")
		return
	}

	userID, err := userIDFromClaims(ctx)
	if err != nil {
		writePublicError(ctx, errcode.ErrTokenInvalid, errors.KindUnauthenticated, "令牌中的用户编号无效")
		return
	}

	if _, err = h.tokenVersions.Bump(ctx.Request.Context(), userID); err != nil {
		writePublicError(ctx, errcode.ErrUnknown, errors.KindInternal, "退出全部登录失败")
		return
	}
	if h.users != nil {
		_, _ = h.users.RevokeStaffUserSessions(ctx.Request.Context(), &upbv1.RevokeStaffUserSessionsRequest{UserId: int32(userID)})
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": true})
}

func (h *staffAuthHandler) Me(ctx *gin.Context) {
	if h == nil || h.users == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "员工认证服务暂不可用")
		return
	}

	userID, err := userIDFromClaims(ctx)
	if err != nil {
		writePublicError(ctx, errcode.ErrTokenInvalid, errors.KindUnauthenticated, "令牌中的用户编号无效")
		return
	}

	authUser, err := h.users.GetUserAuthById(ctx.Request.Context(), &upbv1.IdRequest{Id: int32(userID)})
	if err != nil || authUser == nil || authUser.GetUser() == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "员工资料服务暂不可用")
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
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "紧急授权服务暂不可用")
		return
	}
	var request breakGlassSessionRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeValidationError(ctx, "紧急授权请求参数无效")
		return
	}
	approval, err := h.users.ConsumeBreakGlassApproval(ctx.Request.Context(), &upbv1.ConsumeBreakGlassApprovalRequest{
		ApprovalId:      strings.TrimSpace(request.ApprovalID),
		RequesterUserId: request.RequesterUserID,
		RequestId:       requestID(ctx),
	})
	if err != nil || approval == nil {
		breakGlassEvents.Inc("approval_missing")
		writePublicError(ctx, errcode.ErrPermissionDenied, errors.KindPermissionDenied, "需要有效的紧急授权审批")
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
			Audience:  h.jwtOpts.AudienceValues(),
			ID:        correlationID,
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(timeout)),
			Issuer:    h.jwtOpts.Realm,
		},
	})
	if err != nil {
		writePublicError(ctx, errcode.ErrUnknown, errors.KindInternal, "创建紧急会话失败")
		return
	}
	if err = h.createAdminAuditLog(ctx.Request.Context(), &upbv1.AdminAuditLog{
		TargetUserId:       request.RequesterUserID,
		ActorPrincipalType: string(authz.PrincipalAdminBootstrap),
		Action:             "break_glass_session_issued",
		Detail:             fmt.Sprintf("correlation_id:%s key_id:%s approval_id:%s approver_user_id:%d grants:none", correlationID, keyID, approval.GetId(), approval.GetApproverUserId()),
		CorrelationId:      correlationID,
		RequestId:          requestID(ctx),
		TargetType:         "break_glass_session",
		TargetId:           approval.GetId(),
		Domain:             string(authz.BusinessDomainPlatform),
	}); err != nil {
		breakGlassEvents.Inc("audit_failed")
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "紧急授权审计服务暂不可用")
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

func (h *staffAuthHandler) CreateBreakGlassApproval(ctx *gin.Context) {
	if h == nil || h.users == nil || h.adminAuth == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "紧急授权审批服务暂不可用")
		return
	}
	var request breakGlassApprovalRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeValidationError(ctx, "紧急授权审批请求参数无效")
		return
	}
	ttl := time.Duration(request.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = h.adminAuth.EffectiveBreakGlassTTL()
	}
	approval, err := h.users.CreateBreakGlassApproval(ctx.Request.Context(), &upbv1.CreateBreakGlassApprovalRequest{
		RequesterUserId: request.RequesterUserID,
		Reason:          strings.TrimSpace(request.Reason),
		RequestId:       requestID(ctx),
		ExpiresAt:       uint64(time.Now().Add(ttl).Unix()),
	})
	if err != nil || approval == nil {
		breakGlassEvents.Inc("request_failed")
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "创建紧急授权审批失败")
		return
	}
	_ = h.createAdminAuditLog(ctx.Request.Context(), &upbv1.AdminAuditLog{
		TargetUserId:       request.RequesterUserID,
		ActorPrincipalType: string(authz.PrincipalAdminBootstrap),
		Action:             "break_glass_approval_requested",
		Detail:             fmt.Sprintf("approval_id:%s reason:%s", approval.GetId(), strings.TrimSpace(request.Reason)),
		CorrelationId:      approval.GetId(),
		RequestId:          requestID(ctx),
		TargetType:         "break_glass_approval",
		TargetId:           approval.GetId(),
		Domain:             string(authz.BusinessDomainPlatform),
	})
	breakGlassEvents.Inc("requested")
	ctx.JSON(http.StatusOK, approval)
}

func (h *staffAuthHandler) ApproveBreakGlassApproval(ctx *gin.Context) {
	if h == nil || h.users == nil {
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "紧急授权审批服务暂不可用")
		return
	}
	var request breakGlassApproveRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		writeValidationError(ctx, "紧急授权审批确认参数无效")
		return
	}
	approval, err := h.users.ApproveBreakGlassApproval(ctx.Request.Context(), &upbv1.ApproveBreakGlassApprovalRequest{
		ApprovalId:     strings.TrimSpace(ctx.Param("approval_id")),
		ApproverUserId: request.ApproverUserID,
		RequestId:      requestID(ctx),
	})
	if err != nil || approval == nil {
		breakGlassEvents.Inc("approve_failed")
		writePublicError(ctx, errcode.ErrServiceUnavailable, errors.KindUnavailable, "确认紧急授权审批失败")
		return
	}
	_ = h.createAdminAuditLog(ctx.Request.Context(), &upbv1.AdminAuditLog{
		TargetUserId:       approval.GetRequesterUserId(),
		ActorUserId:        approval.GetApproverUserId(),
		ActorPrincipalType: string(authz.PrincipalAdminBootstrap),
		Action:             "break_glass_approval_approved",
		Detail:             fmt.Sprintf("approval_id:%s", approval.GetId()),
		CorrelationId:      approval.GetId(),
		RequestId:          requestID(ctx),
		TargetType:         "break_glass_approval",
		TargetId:           approval.GetId(),
		Domain:             string(authz.BusinessDomainPlatform),
	})
	breakGlassEvents.Inc("approved")
	ctx.JSON(http.StatusOK, approval)
}

func (h *staffAuthHandler) createToken(ctx context.Context, authUser *upbv1.UserAuthResponse, sessionID string) (string, int64, error) {
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
		SessionID:       sessionID,
		ResourceDomains: effectiveResourceDomains(authUser),
		ResourceStores:  append([]string(nil), authUser.GetResourceStores()...),
		ResourceTeams:   append([]string(nil), authUser.GetResourceTeams()...),
		ResourceScopes:  effectiveResourceScopes(authUser),
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  h.jwtOpts.AudienceValues(),
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

func effectiveResourceScopes(user *upbv1.UserAuthResponse) []string {
	if user == nil {
		return nil
	}
	result := make([]string, 0, len(user.GetResourceScopes()))
	for _, scope := range user.GetResourceScopes() {
		result = append(result, authz.EncodeResourceScope(authz.ResourceScope{
			Domain:       scope.GetDomain(),
			StoreID:      scope.GetStoreId(),
			TeamID:       scope.GetTeamId(),
			ResourceType: scope.GetResourceType(),
			ResourceID:   scope.GetResourceId(),
		}))
	}
	return result
}

func (h *staffAuthHandler) currentTokenVersion(ctx context.Context, userID int32) (uint64, error) {
	if h == nil || h.tokenVersions == nil || userID <= 0 {
		return 0, nil
	}
	return h.tokenVersions.CurrentVersion(ctx, uint64(userID))
}

func (h *staffAuthHandler) createStaffSession(ctx *gin.Context, userID int32) (string, error) {
	if h == nil || h.users == nil || h.jwtOpts == nil {
		return "", status.Error(codes.FailedPrecondition, "staff session backend is not initialized")
	}
	tokenHash := make([]byte, 32)
	if _, err := rand.Read(tokenHash); err != nil {
		return "", err
	}
	deviceID := strings.TrimSpace(ctx.GetHeader("X-Device-ID"))
	if deviceID == "" {
		deviceID = "admin-web"
	}
	deviceName := strings.TrimSpace(ctx.GetHeader("X-Device-Name"))
	if deviceName == "" {
		deviceName = strings.TrimSpace(ctx.GetHeader("User-Agent"))
	}
	if deviceName == "" {
		deviceName = "admin-console"
	}
	session, err := h.users.CreateSession(ctx.Request.Context(), &upbv1.CreateSessionRequest{
		UserId:           userID,
		DeviceId:         deviceID,
		DeviceName:       deviceName,
		RefreshTokenHash: tokenHash,
		ExpiresAt:        uint64(time.Now().Add(h.jwtOpts.Timeout).Unix()),
		PrincipalType:    string(authz.PrincipalStaff),
	})
	if err != nil {
		return "", err
	}
	_, _ = h.users.RecordLogin(ctx.Request.Context(), &upbv1.RecordLoginRequest{UserId: userID, LoggedInAt: uint64(time.Now().Unix())})
	return session.GetId(), nil
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
	return gauth.NewJWTStrategy([]byte(opts.Key), opts.Realm, opts.Audience, middlewares.KeyUserID, func(_ interface{}, c *gin.Context) bool {
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
		if sessionID, _ := claims["session_id"].(string); strings.TrimSpace(sessionID) != "" {
			active, err := users.ValidateSession(c.Request.Context(), &upbv1.ValidateSessionRequest{UserId: int32(userID), SessionId: sessionID})
			if err != nil || active == nil || !active.GetActive() {
				return false
			}
		}
		return true
	}), nil
}

func jwtExpiresAt(ctx *gin.Context) (time.Time, error) {
	exp, ok := gauth.ExtractClaims(ctx)["exp"]
	if !ok {
		return time.Time{}, errors.NewCode(errcode.ErrTokenInvalid, "token missing exp")
	}

	var unix int64
	switch value := exp.(type) {
	case float64:
		unix = int64(value)
	case json.Number:
		v, err := value.Int64()
		if err != nil {
			return time.Time{}, errors.NewCode(errcode.ErrTokenInvalid, "token exp invalid")
		}
		unix = v
	case int64:
		unix = value
	case int:
		unix = int64(value)
	default:
		return time.Time{}, errors.NewCode(errcode.ErrTokenInvalid, "token exp invalid")
	}
	if unix <= 0 {
		return time.Time{}, errors.NewCode(errcode.ErrTokenInvalid, "token exp invalid")
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
	return 0, errors.NewCode(errcode.ErrTokenInvalid, "token user id invalid")
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
