package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	upbv1 "goshop/api/user/v1"
	"goshop/app/goshop/admin/config"
	"goshop/app/pkg/authz"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/restserver"
	"goshop/gmicro/server/restserver/middlewares"
	gauth "goshop/gmicro/server/restserver/middlewares/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestInitRouterRestrictsBootstrapTokenToBreakGlass(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token: "bootstrap-secret",
		},
	}
	client := &fakeAdminUserClient{
		listResponse: &upbv1.UserListResponse{
			Total: 1,
			Data:  []*upbv1.UserInfoResponse{{Id: 1, Username: "staff_001"}},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/user/list", nil)
	req.Header.Set("X-Admin-Token", "bootstrap-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bootstrap header status = %d, want 401", rec.Code)
	}

	breakGlassReq := httptest.NewRequest(http.MethodPost, "/v1/break_glass/session", nil)
	breakGlassReq.Header.Set("X-Admin-Token", "bootstrap-secret")
	breakGlassRec := httptest.NewRecorder()
	server.ServeHTTP(breakGlassRec, breakGlassReq)
	if breakGlassRec.Code != http.StatusOK {
		t.Fatalf("break_glass status = %d, want 200", breakGlassRec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(breakGlassRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode break-glass response: %v", err)
	}
	if _, exists := payload["role"]; exists {
		t.Fatal("break-glass response must not contain a configured role")
	}
	if _, exists := payload["permissions"]; exists {
		t.Fatal("break-glass response must not contain configured permissions")
	}
	tokenText, _ := payload["token"].(string)
	parsed, err := jwt.Parse(tokenText, func(*jwt.Token) (any, error) { return []byte(cfg.Jwt.Key), nil })
	if err != nil || !parsed.Valid {
		t.Fatalf("parse break-glass token: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if roles, exists := claims["roles"]; exists && len(roles.([]any)) != 0 {
		t.Fatalf("break-glass roles = %#v, want none", roles)
	}
	if scope, exists := claims["scope"]; exists && len(scope.([]any)) != 0 {
		t.Fatalf("break-glass scope = %#v, want none", scope)
	}
	if client.createAdminAuditLogReq == nil || client.createAdminAuditLogReq.GetLog().GetAction() != "break_glass_session_issued" {
		t.Fatalf("break_glass audit request = %#v, want action break_glass_session_issued", client.createAdminAuditLogReq)
	}
}

func TestInitRouterAllowsStaffJWTOnUserList(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token: "bootstrap-secret",
		},
	}
	client := &fakeAdminUserClient{
		listResponse: &upbv1.UserListResponse{
			Total: 1,
			Data:  []*upbv1.UserInfoResponse{{Id: 1, Username: "staff_001"}},
		},
		authUserResponse: &upbv1.UserAuthResponse{
			User:       &upbv1.UserInfoResponse{Id: 7, Username: "staff_001", Status: string(authz.AccountStatusActive)},
			LegacyRole: int32(authz.LegacyUserRoleAdmin),
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}

	token, err := middlewares.NewJWT(cfg.Jwt.Key).CreateToken(middlewares.CustomClaims{
		ID:            7,
		AuthorityId:   uint(authz.LegacyUserRoleAdmin),
		Roles:         []string{string(authz.StaffRoleAdmin)},
		PrincipalType: string(authz.PrincipalStaff),
		AccountStatus: string(authz.AccountStatusActive),
		Scope:         []string{string(authz.PermissionUserListAny)},
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    cfg.Jwt.Realm,
		},
	})
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/user/list", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("staff jwt status = %d, want 200", rec.Code)
	}
}

func TestStaffLogoutRevokesCurrentToken(t *testing.T) {
	store := &fakeAdminRevocationStore{}
	handler := newStaffAuthHandler(nil, &options.JwtOptions{}, nil, store, nil)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	ctx.Request.Header.Set("Authorization", "Bearer staff-token")
	ctx.Set(middlewares.JWTPayloadKey, map[string]any{
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	})

	handler.Logout(ctx)

	if !store.revokeCalled {
		t.Fatal("Logout() did not revoke token")
	}
	if store.revokedToken != "staff-token" {
		t.Fatalf("revoked token = %q, want staff-token", store.revokedToken)
	}
}

func TestStaffLogoutAllBumpsTokenVersion(t *testing.T) {
	versions := &fakeAdminTokenVersionStore{}
	handler := newStaffAuthHandler(nil, &options.JwtOptions{}, nil, nil, versions)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/logout_all", nil)
	ctx.Set(middlewares.JWTPayloadKey, map[string]any{
		"user_id": float64(7),
	})

	handler.LogoutAll(ctx)

	if versions.bumpUserID != 7 {
		t.Fatalf("bump user id = %d, want 7", versions.bumpUserID)
	}
}

func TestNewStaffJWTAuthRejectsTokenVersionMismatch(t *testing.T) {
	strategy, err := newStaffJWTAuth(
		&options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		&fakeAdminRevocationStore{},
		&fakeAdminTokenVersionStore{currentVersion: 2},
		&fakeAdminUserClient{authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 7, Status: string(authz.AccountStatusActive)},
		}},
	)
	if err != nil {
		t.Fatalf("newStaffJWTAuth() error = %v", err)
	}
	jwtStrategy := strategy.(gauth.JWTStrategy)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/user/list", nil)
	ctx.Request.Header.Set("Authorization", "Bearer staff-token")
	ctx.Set(middlewares.JWTPayloadKey, map[string]any{
		"user_id":        float64(7),
		"principal_type": string(authz.PrincipalStaff),
		"status":         string(authz.AccountStatusActive),
		"token_version":  float64(1),
	})

	if jwtStrategy.Authorizator(nil, ctx) {
		t.Fatal("Authorizator() = true, want false")
	}
}

func TestStaffMeReturnsCurrentProfile(t *testing.T) {
	handler := newStaffAuthHandler(&fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{
				Id:       7,
				Username: "staff_001",
				Status:   string(authz.AccountStatusActive),
			},
			StaffRoles:  []string{string(authz.StaffRoleAdmin)},
			Permissions: []string{string(authz.PermissionUserReadAny)},
		},
	}, &options.JwtOptions{}, nil, nil, nil)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	ctx.Set(middlewares.JWTPayloadKey, map[string]any{
		"user_id": float64(7),
	})

	handler.Me(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("Me() status = %d, want 200", rec.Code)
	}
}

func TestStaffLoginCreatesAdminAuditLog(t *testing.T) {
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{
				Id:       7,
				Username: "staff_001",
				Status:   string(authz.AccountStatusActive),
			},
			PasswordHash: "hashed",
			StaffRoles:   []string{string(authz.StaffRoleAdmin)},
			Permissions:  []string{string(authz.PermissionUserListAny)},
		},
	}
	handler := newStaffAuthHandler(
		client,
		&options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		&config.AdminAuthOptions{},
		nil,
		&fakeAdminTokenVersionStore{},
	)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"username":"staff_001","password":"Secret123!"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.Login(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("Login() status = %d, want 200", rec.Code)
	}
	if client.createAdminAuditLogReq == nil {
		t.Fatal("Login() did not create admin audit log")
	}
	if client.createAdminAuditLogReq.GetLog().GetAction() != "staff_login_succeeded" {
		t.Fatalf("login audit action = %q, want staff_login_succeeded", client.createAdminAuditLogReq.GetLog().GetAction())
	}
	if client.createAdminAuditLogReq.GetLog().GetTargetUserId() != 7 {
		t.Fatalf("login audit target user id = %d, want 7", client.createAdminAuditLogReq.GetLog().GetTargetUserId())
	}
}

func TestInitRouterAllowsStatusUpdateAndInvalidatesSessions(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 9, Username: "ops_001", Status: string(authz.AccountStatusActive)},
		},
		userResponse: &upbv1.UserInfoResponse{Id: 7, Username: "staff_001", Status: string(authz.AccountStatusActive)},
	}
	versions := &fakeAdminTokenVersionStore{}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, versions); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}

	token, err := middlewares.NewJWT(cfg.Jwt.Key).CreateToken(middlewares.CustomClaims{
		ID:            9,
		AuthorityId:   uint(authz.LegacyUserRoleAdmin),
		Roles:         []string{string(authz.StaffRoleAdmin)},
		PrincipalType: string(authz.PrincipalStaff),
		AccountStatus: string(authz.AccountStatusActive),
		Scope:         []string{string(authz.PermissionUserDisableAny)},
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    cfg.Jwt.Realm,
		},
	})
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/v1/user/7/status", strings.NewReader(`{"status":"disabled"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status update code = %d, want 200", rec.Code)
	}
	if client.updateStatusReq == nil || client.updateStatusReq.GetStatus() != "disabled" {
		t.Fatalf("update status request = %#v, want disabled", client.updateStatusReq)
	}
	if versions.bumpUserID != 7 {
		t.Fatalf("bumped user id = %d, want 7", versions.bumpUserID)
	}
}

func TestInitRouterRejectsSelfDisable(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 9, Username: "admin_001", Status: string(authz.AccountStatusActive)},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleAdmin)}, []string{string(authz.PermissionUserDisableAny)})

	req := httptest.NewRequest(http.MethodPut, "/v1/user/9/status", strings.NewReader(`{"status":"disabled"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("self disable status = %d, want 400", rec.Code)
	}
}

func TestInitRouterRejectsCreateSuperAdminForNonSuperAdmin(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 9, Username: "admin_001", Status: string(authz.AccountStatusActive)},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleAdmin)}, []string{string(authz.PermissionUserCreateAny)})

	req := httptest.NewRequest(http.MethodPost, "/v1/user/staff", strings.NewReader(`{"mobile":"13800138000","password":"Secret123!","roles":["super_admin"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("create super admin status = %d, want 403", rec.Code)
	}
}

func TestInitRouterAllowsCreateStaff(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 9, Username: "admin_001", Status: string(authz.AccountStatusActive)},
		},
		createStaffResponse: &upbv1.StaffUserResponse{
			User:        &upbv1.UserInfoResponse{Id: 11, Username: "ops_001", Status: string(authz.AccountStatusActive)},
			Roles:       []string{string(authz.StaffRoleOps)},
			Permissions: []string{string(authz.PermissionOrderReadAny)},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleAdmin)}, []string{string(authz.PermissionUserCreateAny)})

	req := httptest.NewRequest(http.MethodPost, "/v1/user/staff", strings.NewReader(`{"username":"ops_001","mobile":"13800138000","email":"ops@example.com","nick_name":"ops","password":"Secret123!","roles":["ops"],"status":"active"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("create staff status = %d, want 200", rec.Code)
	}
	if client.createStaffReq == nil || client.createStaffReq.GetUser().GetUsername() != "ops_001" {
		t.Fatalf("create staff request = %#v, want username ops_001", client.createStaffReq)
	}
	if client.createStaffReq.GetActor().GetActorUserId() != 9 {
		t.Fatalf("create staff actor = %#v, want actor user id 9", client.createStaffReq.GetActor())
	}
}

func TestInitRouterRejectsCrossDomainCreateStaff(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 9, Username: "ops_001", Status: string(authz.AccountStatusActive)},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleOps)}, []string{string(authz.PermissionUserCreateAny)})

	req := httptest.NewRequest(http.MethodPost, "/v1/user/staff", strings.NewReader(`{"username":"finance_001","mobile":"13800138001","password":"Secret123!","roles":["finance"],"status":"active"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-domain create status = %d, want 403", rec.Code)
	}
	if client.createStaffReq != nil {
		t.Fatalf("cross-domain create should not reach rpc, got %#v", client.createStaffReq)
	}
}

func TestInitRouterRejectsHighRiskWriteWithoutConfirmation(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 9, Username: "admin_001", Status: string(authz.AccountStatusActive)},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleAdmin)}, []string{string(authz.PermissionUserCreateAny)})

	req := httptest.NewRequest(http.MethodPost, "/v1/user/staff", strings.NewReader(`{"mobile":"13800138000","password":"Secret123!","roles":["ops"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing confirmation status = %d, want 403", rec.Code)
	}
}

func TestInitRouterPassesAuditLogFilters(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token: "bootstrap-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 9, Username: "admin_001", Status: string(authz.AccountStatusActive)},
		},
		auditLogsResponse: &upbv1.UserAuditLogListResponse{
			Total: 1,
			Data: []*upbv1.UserAuditLog{{
				Id:          1,
				UserId:      7,
				Action:      "staff_user_status_updated",
				ActorUserId: 9,
			}},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleAdmin)}, []string{string(authz.PermissionAuditReadAny)})

	req := httptest.NewRequest(http.MethodGet, "/v1/user/7/audit_logs?action=staff_user_status_updated&actor_user_id=9&actor_principal_type=staff&created_after=1700000000&created_before=1700003600", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("audit logs status = %d, want 200", rec.Code)
	}
	if client.auditLogsReq == nil {
		t.Fatal("audit logs request was not captured")
	}
	if client.auditLogsReq.GetAction() != "staff_user_status_updated" {
		t.Fatalf("audit action = %q, want staff_user_status_updated", client.auditLogsReq.GetAction())
	}
	if client.auditLogsReq.GetActorUserId() != 9 {
		t.Fatalf("audit actor user id = %d, want 9", client.auditLogsReq.GetActorUserId())
	}
	if client.auditLogsReq.GetActorPrincipalType() != "staff" {
		t.Fatalf("audit actor principal type = %q, want staff", client.auditLogsReq.GetActorPrincipalType())
	}
	if client.auditLogsReq.GetCreatedAfter() != 1700000000 || client.auditLogsReq.GetCreatedBefore() != 1700003600 {
		t.Fatalf("audit time range = (%d, %d), want (1700000000, 1700003600)", client.auditLogsReq.GetCreatedAfter(), client.auditLogsReq.GetCreatedBefore())
	}
}

func TestInitRouterPassesAdminAuditLogFilters(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token: "bootstrap-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 9, Username: "admin_001", Status: string(authz.AccountStatusActive)},
		},
		adminAuditLogsResponse: &upbv1.AdminAuditLogListResponse{
			Total: 1,
			Data: []*upbv1.AdminAuditLog{{
				Id:                 1,
				TargetUserId:       7,
				Action:             "staff_login_succeeded",
				ActorUserId:        9,
				ActorPrincipalType: string(authz.PrincipalStaff),
			}},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleAdmin)}, []string{string(authz.PermissionAuditReadAny)})

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/audit_logs?target_user_id=7&action=staff_login_succeeded&actor_user_id=9&actor_principal_type=staff&created_after=1700000000&created_before=1700003600", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("admin audit logs status = %d, want 200", rec.Code)
	}
	if client.adminAuditLogsReq == nil {
		t.Fatal("admin audit logs request was not captured")
	}
	if client.adminAuditLogsReq.GetTargetUserId() != 7 {
		t.Fatalf("admin audit target user id = %d, want 7", client.adminAuditLogsReq.GetTargetUserId())
	}
	if client.adminAuditLogsReq.GetAction() != "staff_login_succeeded" {
		t.Fatalf("admin audit action = %q, want staff_login_succeeded", client.adminAuditLogsReq.GetAction())
	}
	if client.adminAuditLogsReq.GetActorUserId() != 9 {
		t.Fatalf("admin audit actor user id = %d, want 9", client.adminAuditLogsReq.GetActorUserId())
	}
	if client.adminAuditLogsReq.GetActorPrincipalType() != "staff" {
		t.Fatalf("admin audit actor principal type = %q, want staff", client.adminAuditLogsReq.GetActorPrincipalType())
	}
	if client.adminAuditLogsReq.GetCreatedAfter() != 1700000000 || client.adminAuditLogsReq.GetCreatedBefore() != 1700003600 {
		t.Fatalf("admin audit time range = (%d, %d), want (1700000000, 1700003600)", client.adminAuditLogsReq.GetCreatedAfter(), client.adminAuditLogsReq.GetCreatedBefore())
	}
}

func TestInitRouterListsPermissionTemplates(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token: "bootstrap-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 9, Username: "ops_001", Status: string(authz.AccountStatusActive)},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleOps)}, []string{string(authz.PermissionRoleReadAny)})

	req := httptest.NewRequest(http.MethodGet, "/v1/staff/permission_templates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("permission templates status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"name":"ops"`) {
		t.Fatalf("permission templates body missing ops template: %s", body)
	}
	if !strings.Contains(body, `"name":"finance"`) {
		t.Fatalf("permission templates body missing finance template: %s", body)
	}
	if !strings.Contains(body, `"manageable":false`) {
		t.Fatalf("permission templates body missing unmanageable template marker: %s", body)
	}
}

func TestInitRouterRejectsCrossDomainRoleAssignment(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{Id: 9, Username: "ops_001", Status: string(authz.AccountStatusActive)},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleOps)}, []string{string(authz.PermissionRoleAssignAny)})

	req := httptest.NewRequest(http.MethodPut, "/v1/user/7/roles", strings.NewReader(`{"roles":["finance"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-domain role assignment status = %d, want 403", rec.Code)
	}
	if client.replaceRolesReq != nil {
		t.Fatalf("cross-domain role assignment should not reach rpc, got %#v", client.replaceRolesReq)
	}
}

func TestInitRouterUpdatesStaffRole(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User:       &upbv1.UserInfoResponse{Id: 9, Username: "root_001", Status: string(authz.AccountStatusActive)},
			LegacyRole: int32(authz.LegacyUserRoleAdmin),
		},
		updateRoleResp: &upbv1.StaffRole{
			Name:        string(authz.StaffRoleOps),
			Description: "updated ops role",
			Permissions: []string{string(authz.PermissionOrderCloseAny), string(authz.PermissionOrderReadAny)},
			Builtin:     true,
			Domains:     []string{string(authz.BusinessDomainOps)},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleSuperAdmin)}, []string{
		string(authz.PermissionRoleWriteAny),
		string(authz.PermissionOrderCloseAny),
		string(authz.PermissionOrderReadAny),
	})

	req := httptest.NewRequest(http.MethodPut, "/v1/staff/roles/ops", strings.NewReader(`{"description":"updated ops role","permissions":["order:close:any","order:read:any"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("update staff role status = %d, want 200", rec.Code)
	}
	if client.updateRoleReq == nil || client.updateRoleReq.GetRole().GetName() != string(authz.StaffRoleOps) {
		t.Fatalf("update role request = %#v, want role ops", client.updateRoleReq)
	}
}

func TestInitRouterRejectsRolePermissionEscalation(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User:       &upbv1.UserInfoResponse{Id: 9, Username: "root_001", Status: string(authz.AccountStatusActive)},
			LegacyRole: int32(authz.LegacyUserRoleAdmin),
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleSuperAdmin)}, []string{
		string(authz.PermissionRoleWriteAny),
		string(authz.PermissionOrderReadAny),
	})

	req := httptest.NewRequest(http.MethodPut, "/v1/staff/roles/ops", strings.NewReader(`{"description":"updated ops role","permissions":["order:read:any","order:refund:any"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("permission escalation status = %d, want 403", rec.Code)
	}
	if client.updateRoleReq != nil {
		t.Fatalf("permission escalation should not reach rpc, got %#v", client.updateRoleReq)
	}
}

func TestInitRouterCreatesCustomStaffRole(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User:       &upbv1.UserInfoResponse{Id: 9, Username: "root_001", Status: string(authz.AccountStatusActive)},
			LegacyRole: int32(authz.LegacyUserRoleAdmin),
		},
		createRoleResp: &upbv1.StaffRole{
			Name:        "ops_delegate",
			Description: "operations delegate",
			Permissions: []string{string(authz.PermissionOrderReadAny)},
			Domains:     []string{string(authz.BusinessDomainOps)},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleOps)}, []string{
		string(authz.PermissionRoleWriteAny),
		string(authz.PermissionOrderReadAny),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/staff/roles", strings.NewReader(`{"name":"ops_delegate","description":"operations delegate","permissions":["order:read:any"],"domains":["operations"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("create custom role status = %d, want 200", rec.Code)
	}
	if client.createRoleReq == nil || client.createRoleReq.GetRole().GetName() != "ops_delegate" {
		t.Fatalf("create role request = %#v, want ops_delegate", client.createRoleReq)
	}
}

func TestInitRouterRejectsCrossDomainCustomStaffRoleCreation(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User:       &upbv1.UserInfoResponse{Id: 9, Username: "ops_001", Status: string(authz.AccountStatusActive)},
			LegacyRole: int32(authz.LegacyUserRoleAdmin),
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleOps)}, []string{
		string(authz.PermissionRoleWriteAny),
		string(authz.PermissionOrderReadAny),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/staff/roles", strings.NewReader(`{"name":"finance_delegate","description":"finance delegate","permissions":["order:read:any"],"domains":["finance"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-domain custom role create status = %d, want 403", rec.Code)
	}
	if client.createRoleReq != nil {
		t.Fatalf("cross-domain custom role create should not reach rpc, got %#v", client.createRoleReq)
	}
}

func TestInitRouterDeletesCustomStaffRole(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User:       &upbv1.UserInfoResponse{Id: 9, Username: "ops_001", Status: string(authz.AccountStatusActive)},
			LegacyRole: int32(authz.LegacyUserRoleAdmin),
		},
		staffRolesResponse: &upbv1.StaffRoleListResponse{
			Roles: []*upbv1.StaffRole{
				{Name: "ops_delegate", Domains: []string{string(authz.BusinessDomainOps)}},
			},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleOps)}, []string{
		string(authz.PermissionRoleWriteAny),
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/staff/roles/ops_delegate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("delete custom role status = %d, want 200", rec.Code)
	}
	if client.deleteRoleReq == nil || client.deleteRoleReq.GetName() != "ops_delegate" {
		t.Fatalf("delete role request = %#v, want ops_delegate", client.deleteRoleReq)
	}
}

func TestInitRouterRejectsCrossDomainCustomStaffRoleDeletion(t *testing.T) {
	server := restserver.NewServer()
	cfg := &config.Config{
		Server: options.NewServerOptions(),
		Jwt:    &options.JwtOptions{Realm: "admin", Key: "01234567890123456789012345678901", Timeout: time.Hour, MaxRefresh: time.Hour},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
	client := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User:       &upbv1.UserInfoResponse{Id: 9, Username: "ops_001", Status: string(authz.AccountStatusActive)},
			LegacyRole: int32(authz.LegacyUserRoleAdmin),
		},
		staffRolesResponse: &upbv1.StaffRoleListResponse{
			Roles: []*upbv1.StaffRole{
				{Name: "finance_delegate", Domains: []string{string(authz.BusinessDomainFinance)}},
			},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, client, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouter() error = %v", err)
	}
	token := mustCreateAdminToken(t, cfg.Jwt, 9, []string{string(authz.StaffRoleOps)}, []string{
		string(authz.PermissionRoleWriteAny),
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/staff/roles/finance_delegate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-domain custom role delete status = %d, want 403", rec.Code)
	}
	if client.deleteRoleReq != nil {
		t.Fatalf("cross-domain custom role delete should not reach rpc, got %#v", client.deleteRoleReq)
	}
}

func mustCreateAdminToken(t *testing.T, jwtOpts *options.JwtOptions, userID uint, roles, scope []string) string {
	t.Helper()
	token, err := middlewares.NewJWT(jwtOpts.Key).CreateToken(middlewares.CustomClaims{
		ID:            userID,
		AuthorityId:   uint(authz.LegacyUserRoleAdmin),
		Roles:         append([]string(nil), roles...),
		PrincipalType: string(authz.PrincipalStaff),
		AccountStatus: string(authz.AccountStatusActive),
		Scope:         append([]string(nil), scope...),
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    jwtOpts.Realm,
		},
	})
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}
	return token
}

type fakeAdminUserClient struct {
	upbv1.UserClient
	listResponse           *upbv1.UserListResponse
	userResponse           *upbv1.UserInfoResponse
	authUserResponse       *upbv1.UserAuthResponse
	updateStatusReq        *upbv1.UpdateUserStatusRequest
	createRoleReq          *upbv1.CreateStaffRoleRequest
	createRoleResp         *upbv1.StaffRole
	updateRoleReq          *upbv1.UpdateStaffRoleRequest
	updateRoleResp         *upbv1.StaffRole
	deleteRoleReq          *upbv1.DeleteStaffRoleRequest
	createAdminAuditLogReq *upbv1.CreateAdminAuditLogRequest
	createStaffReq         *upbv1.CreateStaffUserRequest
	createStaffResponse    *upbv1.StaffUserResponse
	replaceRolesReq        *upbv1.ReplaceUserStaffRolesRequest
	replaceRolesResp       *upbv1.UserRoleBindingResponse
	staffRolesResponse     *upbv1.StaffRoleListResponse
	auditLogsResponse      *upbv1.UserAuditLogListResponse
	auditLogsReq           *upbv1.UserAuditLogPageRequest
	adminAuditLogsResponse *upbv1.AdminAuditLogListResponse
	adminAuditLogsReq      *upbv1.AdminAuditLogPageRequest
}

func (f *fakeAdminUserClient) GetUserList(context.Context, *upbv1.PageInfo, ...grpc.CallOption) (*upbv1.UserListResponse, error) {
	if f.listResponse != nil {
		return f.listResponse, nil
	}
	return &upbv1.UserListResponse{}, nil
}

func (f *fakeAdminUserClient) GetUserByMobile(context.Context, *upbv1.MobileRequest, ...grpc.CallOption) (*upbv1.UserInfoResponse, error) {
	return &upbv1.UserInfoResponse{}, nil
}

func (f *fakeAdminUserClient) GetUserById(context.Context, *upbv1.IdRequest, ...grpc.CallOption) (*upbv1.UserInfoResponse, error) {
	if f.userResponse != nil {
		return f.userResponse, nil
	}
	return &upbv1.UserInfoResponse{}, nil
}

func (f *fakeAdminUserClient) UpdateUserStatus(_ context.Context, req *upbv1.UpdateUserStatusRequest, _ ...grpc.CallOption) (*upbv1.UserInfoResponse, error) {
	f.updateStatusReq = req
	if f.userResponse != nil {
		return &upbv1.UserInfoResponse{
			Id:       f.userResponse.GetId(),
			Mobile:   f.userResponse.GetMobile(),
			NickName: f.userResponse.GetNickName(),
			BirthDay: f.userResponse.GetBirthDay(),
			Gender:   f.userResponse.GetGender(),
			Email:    f.userResponse.GetEmail(),
			Username: f.userResponse.GetUsername(),
			Status:   req.GetStatus(),
		}, nil
	}
	return &upbv1.UserInfoResponse{Id: req.GetId(), Status: req.GetStatus()}, nil
}

func (f *fakeAdminUserClient) GetUserAuthByMobile(context.Context, *upbv1.MobileRequest, ...grpc.CallOption) (*upbv1.UserAuthResponse, error) {
	if f.authUserResponse != nil {
		return f.authUserResponse, nil
	}
	return &upbv1.UserAuthResponse{}, nil
}

func (f *fakeAdminUserClient) GetUserAuthById(context.Context, *upbv1.IdRequest, ...grpc.CallOption) (*upbv1.UserAuthResponse, error) {
	if f.authUserResponse != nil {
		return f.authUserResponse, nil
	}
	return &upbv1.UserAuthResponse{}, nil
}

func (f *fakeAdminUserClient) ListStaffRoles(context.Context, *emptypb.Empty, ...grpc.CallOption) (*upbv1.StaffRoleListResponse, error) {
	if f.staffRolesResponse != nil {
		return f.staffRolesResponse, nil
	}
	return &upbv1.StaffRoleListResponse{}, nil
}

func (f *fakeAdminUserClient) CreateStaffRole(_ context.Context, req *upbv1.CreateStaffRoleRequest, _ ...grpc.CallOption) (*upbv1.StaffRole, error) {
	f.createRoleReq = req
	if f.createRoleResp != nil {
		return f.createRoleResp, nil
	}
	return &upbv1.StaffRole{}, nil
}

func (f *fakeAdminUserClient) UpdateStaffRole(_ context.Context, req *upbv1.UpdateStaffRoleRequest, _ ...grpc.CallOption) (*upbv1.StaffRole, error) {
	f.updateRoleReq = req
	if f.updateRoleResp != nil {
		return f.updateRoleResp, nil
	}
	return &upbv1.StaffRole{}, nil
}

func (f *fakeAdminUserClient) DeleteStaffRole(_ context.Context, req *upbv1.DeleteStaffRoleRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	f.deleteRoleReq = req
	return &emptypb.Empty{}, nil
}

func (f *fakeAdminUserClient) GetUserStaffRoles(context.Context, *upbv1.IdRequest, ...grpc.CallOption) (*upbv1.UserRoleBindingResponse, error) {
	if f.authUserResponse != nil {
		return &upbv1.UserRoleBindingResponse{
			UserId:      f.authUserResponse.GetUser().GetId(),
			Roles:       append([]string(nil), f.authUserResponse.GetStaffRoles()...),
			Permissions: append([]string(nil), f.authUserResponse.GetPermissions()...),
		}, nil
	}
	return &upbv1.UserRoleBindingResponse{}, nil
}

func (f *fakeAdminUserClient) ReplaceUserStaffRoles(_ context.Context, req *upbv1.ReplaceUserStaffRolesRequest, _ ...grpc.CallOption) (*upbv1.UserRoleBindingResponse, error) {
	f.replaceRolesReq = req
	if f.replaceRolesResp != nil {
		return f.replaceRolesResp, nil
	}
	return &upbv1.UserRoleBindingResponse{}, nil
}

func (f *fakeAdminUserClient) ListUserAuditLogs(_ context.Context, req *upbv1.UserAuditLogPageRequest, _ ...grpc.CallOption) (*upbv1.UserAuditLogListResponse, error) {
	f.auditLogsReq = req
	if f.auditLogsResponse != nil {
		return f.auditLogsResponse, nil
	}
	return &upbv1.UserAuditLogListResponse{}, nil
}

func (f *fakeAdminUserClient) CreateAdminAuditLog(_ context.Context, req *upbv1.CreateAdminAuditLogRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	f.createAdminAuditLogReq = req
	return &emptypb.Empty{}, nil
}

func (f *fakeAdminUserClient) ListAdminAuditLogs(_ context.Context, req *upbv1.AdminAuditLogPageRequest, _ ...grpc.CallOption) (*upbv1.AdminAuditLogListResponse, error) {
	f.adminAuditLogsReq = req
	if f.adminAuditLogsResponse != nil {
		return f.adminAuditLogsResponse, nil
	}
	return &upbv1.AdminAuditLogListResponse{}, nil
}

func (f *fakeAdminUserClient) CreateUser(context.Context, *upbv1.CreateUserInfo, ...grpc.CallOption) (*upbv1.UserInfoResponse, error) {
	return &upbv1.UserInfoResponse{}, nil
}

func (f *fakeAdminUserClient) CreateStaffUser(_ context.Context, req *upbv1.CreateStaffUserRequest, _ ...grpc.CallOption) (*upbv1.StaffUserResponse, error) {
	f.createStaffReq = req
	if f.createStaffResponse != nil {
		return f.createStaffResponse, nil
	}
	return &upbv1.StaffUserResponse{}, nil
}

func (f *fakeAdminUserClient) UpdateUser(context.Context, *upbv1.UpdateUserInfo, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (f *fakeAdminUserClient) DeleteUser(context.Context, *upbv1.IdRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (f *fakeAdminUserClient) CheckPassWord(context.Context, *upbv1.PasswordCheckInfo, ...grpc.CallOption) (*upbv1.CheckResponse, error) {
	return &upbv1.CheckResponse{Success: true}, nil
}

var _ upbv1.UserClient = &fakeAdminUserClient{}

type fakeAdminRevocationStore struct {
	revoked      bool
	revokeCalled bool
	revokedToken string
}

func (f *fakeAdminRevocationStore) Revoke(_ context.Context, token string, _ time.Time) error {
	f.revokeCalled = true
	f.revokedToken = token
	return nil
}
func (f *fakeAdminRevocationStore) IsRevoked(context.Context, string) (bool, error) {
	return f.revoked, nil
}

type fakeAdminTokenVersionStore struct {
	currentVersion uint64
	bumpUserID     uint64
}

func (f *fakeAdminTokenVersionStore) CurrentVersion(context.Context, uint64) (uint64, error) {
	return f.currentVersion, nil
}

func (f *fakeAdminTokenVersionStore) Bump(_ context.Context, userID uint64) (uint64, error) {
	f.bumpUserID = userID
	f.currentVersion++
	return f.currentVersion, nil
}
