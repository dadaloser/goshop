package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	goodspb "goshop/api/goods/v1"
	rpb "goshop/api/review/v1"
	upbv1 "goshop/api/user/v1"
	"goshop/app/goshop/admin/config"
	"goshop/app/pkg/authz"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/restserver"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/common/core"
	"goshop/pkg/errcode"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
)

type adminRouteMatrixSpec struct {
	method            string
	routePath         string
	requestPath       string
	permission        authz.Permission
	roles             []string
	resourceDomain    string
	resourceStoreID   string
	resourceTeamID    string
	needsConfirmation bool
	noPermissionOK    bool
}

func TestAdminRouteMatrixCoversAllBusinessRoutes(t *testing.T) {
	server := newAdminRBACRouteTestServer(t)

	expected := make(map[string]struct{}, len(adminRouteMatrix))
	for _, spec := range adminRouteMatrix {
		expected[spec.method+" "+spec.routePath] = struct{}{}
	}

	for _, route := range server.Routes() {
		if !strings.HasPrefix(route.Path, "/v1/") {
			continue
		}
		key := route.Method + " " + route.Path
		if _, ok := expected[key]; !ok {
			t.Fatalf("route %s is missing from RBAC route matrix", key)
		}
		delete(expected, key)
	}

	if len(expected) > 0 {
		t.Fatalf("RBAC route matrix contains stale routes: %v", mapsKeys(expected))
	}
}

func TestAdminRouteMatrixProtectedRoutesRequireDeclaredPermission(t *testing.T) {
	server := newAdminRBACRouteTestServer(t)
	cfg := newAdminRBACRouteTestConfig()

	for _, spec := range adminRouteMatrix {
		if spec.noPermissionOK || spec.permission == "" {
			continue
		}

		t.Run(spec.method+" "+spec.routePath, func(t *testing.T) {
			req := httptest.NewRequest(spec.method, spec.requestPath, nil)
			req.Header.Set("Authorization", "Bearer "+mustCreateScopedAdminToken(t, cfg.Jwt, 99, spec.roles, nil, spec.resourceDomain, spec.resourceStoreID, spec.resourceTeamID))
			applyAdminRBACRouteHeaders(req, spec)

			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden && rec.Code != http.StatusUnauthorized {
				t.Fatalf("route %s without permission status = %d, want 401 or 403", spec.routePath, rec.Code)
			}
			if rec.Code == http.StatusForbidden {
				var response core.ErrResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("route %s without permission decode error response = %v", spec.routePath, err)
				}
				if response.Code != errcode.ErrPermissionDenied {
					t.Fatalf("route %s without permission error code = %d, want %d", spec.routePath, response.Code, errcode.ErrPermissionDenied)
				}
			}

			authorized := httptest.NewRequest(spec.method, spec.requestPath, nil)
			authorized.Header.Set("Authorization", "Bearer "+mustCreateScopedAdminToken(t, cfg.Jwt, 99, spec.roles, []string{string(spec.permission)}, spec.resourceDomain, spec.resourceStoreID, spec.resourceTeamID))
			applyAdminRBACRouteHeaders(authorized, spec)

			authorizedRec := httptest.NewRecorder()
			server.ServeHTTP(authorizedRec, authorized)
			if authorizedRec.Code == http.StatusForbidden {
				var response core.ErrResponse
				if err := json.Unmarshal(authorizedRec.Body.Bytes(), &response); err != nil {
					t.Fatalf("route %s with permission decode error response = %v", spec.routePath, err)
				}
				if response.Code == errcode.ErrPermissionDenied {
					t.Fatalf("route %s with permission still denied by permission middleware", spec.routePath)
				}
			}
		})
	}
}

func applyAdminRBACRouteHeaders(req *http.Request, spec adminRouteMatrixSpec) {
	if spec.resourceDomain != "" {
		req.Header.Set("X-Resource-Domain", spec.resourceDomain)
	}
	if spec.resourceStoreID != "" {
		req.Header.Set("X-Store-ID", spec.resourceStoreID)
	}
	if spec.resourceTeamID != "" {
		req.Header.Set("X-Team-ID", spec.resourceTeamID)
	}
	if spec.needsConfirmation {
		req.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	}
}

func newAdminRBACRouteTestServer(t *testing.T) *restserver.Server {
	t.Helper()

	server := restserver.NewServer()
	cfg := newAdminRBACRouteTestConfig()
	userClient := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{
				Id:       99,
				Username: "rbac_admin",
				Status:   string(authz.AccountStatusActive),
			},
			LegacyRole: int32(authz.LegacyUserRoleAdmin),
		},
		staffRolesResponse: &upbv1.StaffRoleListResponse{
			Roles: []*upbv1.StaffRole{
				{Name: "ops_delegate", Domains: []string{string(authz.BusinessDomainOps)}},
			},
		},
	}
	if err := initRouterWithSessionStores(server, cfg, userClient, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouterWithSessionStores() error = %v", err)
	}
	if err := registerAdminReviewRoutesWithStores(server, cfg, userClient, &fakeAdminGoodsClient{}, &fakeAdminReviewClient{}, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("registerAdminReviewRoutesWithStores() error = %v", err)
	}
	return server
}

func newAdminRBACRouteTestConfig() *config.Config {
	return &config.Config{
		Server: options.NewServerOptions(),
		Jwt: &options.JwtOptions{
			Realm:      "admin",
			Key:        "01234567890123456789012345678901",
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
		},
		AdminAuth: &config.AdminAuthOptions{
			Token:             "bootstrap-secret",
			ConfirmationToken: "confirm-secret",
		},
	}
}

func mustCreateScopedAdminToken(
	t *testing.T,
	jwtOpts *options.JwtOptions,
	userID uint,
	roles []string,
	scope []string,
	resourceDomain string,
	resourceStoreID string,
	resourceTeamID string,
) string {
	t.Helper()
	return mustCreateAdminTokenWithClaims(t, jwtOpts, middlewares.CustomClaims{
		ID:              userID,
		AuthorityId:     uint(authz.LegacyUserRoleAdmin),
		Roles:           append([]string(nil), roles...),
		PrincipalType:   string(authz.PrincipalStaff),
		AccountStatus:   string(authz.AccountStatusActive),
		Scope:           append([]string(nil), scope...),
		ResourceDomains: collectNonEmpty(resourceDomain),
		ResourceStores:  collectNonEmpty(resourceStoreID),
		ResourceTeams:   collectNonEmpty(resourceTeamID),
	})
}

func mustCreateScopedAdminTokenWithResourceScopes(
	t *testing.T,
	jwtOpts *options.JwtOptions,
	userID uint,
	roles []string,
	scope []string,
	resourceScopes []authz.ResourceScope,
	resourceDomain string,
	resourceStoreID string,
	resourceTeamID string,
) string {
	t.Helper()
	encodedScopes := make([]string, 0, len(resourceScopes))
	for _, resourceScope := range resourceScopes {
		encodedScopes = append(encodedScopes, authz.EncodeResourceScope(resourceScope))
	}
	return mustCreateAdminTokenWithClaims(t, jwtOpts, middlewares.CustomClaims{
		ID:              userID,
		AuthorityId:     uint(authz.LegacyUserRoleAdmin),
		Roles:           append([]string(nil), roles...),
		PrincipalType:   string(authz.PrincipalStaff),
		AccountStatus:   string(authz.AccountStatusActive),
		Scope:           append([]string(nil), scope...),
		ResourceDomains: collectNonEmpty(resourceDomain),
		ResourceStores:  collectNonEmpty(resourceStoreID),
		ResourceTeams:   collectNonEmpty(resourceTeamID),
		ResourceScopes:  encodedScopes,
	})
}

func mustCreateAdminTokenWithClaims(
	t *testing.T,
	jwtOpts *options.JwtOptions,
	claims middlewares.CustomClaims,
) string {
	t.Helper()
	if claims.PrincipalType == "" {
		claims.PrincipalType = string(authz.PrincipalStaff)
	}
	if claims.AccountStatus == "" {
		claims.AccountStatus = string(authz.AccountStatusActive)
	}
	token, err := middlewares.NewJWT(jwtOpts.Key).CreateToken(middlewares.CustomClaims{
		ID:              claims.ID,
		NickName:        claims.NickName,
		AuthorityId:     claims.AuthorityId,
		Roles:           append([]string(nil), claims.Roles...),
		PrincipalType:   claims.PrincipalType,
		AccountStatus:   claims.AccountStatus,
		Scope:           append([]string(nil), claims.Scope...),
		ResourceDomains: append([]string(nil), claims.ResourceDomains...),
		ResourceStores:  append([]string(nil), claims.ResourceStores...),
		ResourceTeams:   append([]string(nil), claims.ResourceTeams...),
		ResourceScopes:  append([]string(nil), claims.ResourceScopes...),
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

func collectNonEmpty(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{value}
}

func mapsKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	return result
}

type fakeAdminReviewClient struct {
	rpb.ReviewClient
}

type fakeAdminGoodsClient struct {
	goodspb.GoodsClient
}

func (f *fakeAdminGoodsClient) GetGoodsDetail(context.Context, *goodspb.GoodInfoRequest, ...grpc.CallOption) (*goodspb.GoodsInfoResponse, error) {
	return &goodspb.GoodsInfoResponse{Id: 1, StoreId: "store-a"}, nil
}

func (f *fakeAdminReviewClient) CreateReview(context.Context, *rpb.CreateReviewRequest, ...grpc.CallOption) (*rpb.ReviewResponse, error) {
	return &rpb.ReviewResponse{}, nil
}

func (f *fakeAdminReviewClient) AppendReview(context.Context, *rpb.AppendReviewRequest, ...grpc.CallOption) (*rpb.ReviewResponse, error) {
	return &rpb.ReviewResponse{}, nil
}

func (f *fakeAdminReviewClient) GetReview(context.Context, *rpb.GetReviewRequest, ...grpc.CallOption) (*rpb.ReviewResponse, error) {
	return &rpb.ReviewResponse{Id: 1, GoodsId: 1}, nil
}

func (f *fakeAdminReviewClient) ListReviews(context.Context, *rpb.ListReviewsRequest, ...grpc.CallOption) (*rpb.ReviewListResponse, error) {
	return &rpb.ReviewListResponse{}, nil
}

func (f *fakeAdminReviewClient) ModerateReview(context.Context, *rpb.ModerateReviewRequest, ...grpc.CallOption) (*rpb.ReviewResponse, error) {
	return &rpb.ReviewResponse{}, nil
}

func (f *fakeAdminReviewClient) ReplyReview(context.Context, *rpb.ReplyReviewRequest, ...grpc.CallOption) (*rpb.ReviewResponse, error) {
	return &rpb.ReviewResponse{}, nil
}

func (f *fakeAdminReviewClient) GetRating(context.Context, *rpb.GetRatingRequest, ...grpc.CallOption) (*rpb.RatingResponse, error) {
	return &rpb.RatingResponse{}, nil
}

func (f *fakeAdminReviewClient) RebuildRating(context.Context, *rpb.RebuildRatingRequest, ...grpc.CallOption) (*rpb.RatingResponse, error) {
	return &rpb.RatingResponse{}, nil
}

var _ rpb.ReviewClient = (*fakeAdminReviewClient)(nil)

var adminRouteMatrix = []adminRouteMatrixSpec{
	{method: http.MethodPost, routePath: "/v1/auth/login", requestPath: "/v1/auth/login", noPermissionOK: true},
	{method: http.MethodPost, routePath: "/v1/auth/logout", requestPath: "/v1/auth/logout", noPermissionOK: true},
	{method: http.MethodPost, routePath: "/v1/auth/logout_all", requestPath: "/v1/auth/logout_all", noPermissionOK: true},
	{method: http.MethodGet, routePath: "/v1/auth/me", requestPath: "/v1/auth/me", noPermissionOK: true},
	{method: http.MethodPost, routePath: "/v1/break_glass/approvals", requestPath: "/v1/break_glass/approvals", noPermissionOK: true},
	{method: http.MethodPost, routePath: "/v1/break_glass/approvals/:approval_id/approve", requestPath: "/v1/break_glass/approvals/approval-1/approve", noPermissionOK: true},
	{method: http.MethodPost, routePath: "/v1/break_glass/session", requestPath: "/v1/break_glass/session", noPermissionOK: true},
	{method: http.MethodGet, routePath: "/v1/admin/audit_logs", requestPath: "/v1/admin/audit_logs", permission: authz.PermissionAuditReadAny, roles: []string{string(authz.StaffRoleAdmin)}},
	{method: http.MethodPost, routePath: "/v1/user/staff", requestPath: "/v1/user/staff", permission: authz.PermissionUserCreateAny, roles: []string{string(authz.StaffRoleAdmin)}, needsConfirmation: true},
	{method: http.MethodGet, routePath: "/v1/user/list", requestPath: "/v1/user/list", permission: authz.PermissionUserListAny, roles: []string{string(authz.StaffRoleAdmin)}},
	{method: http.MethodGet, routePath: "/v1/user/:id", requestPath: "/v1/user/1", permission: authz.PermissionUserReadAny, roles: []string{string(authz.StaffRoleAdmin)}},
	{method: http.MethodPut, routePath: "/v1/user/:id/status", requestPath: "/v1/user/1/status", permission: authz.PermissionUserDisableAny, roles: []string{string(authz.StaffRoleAdmin)}, needsConfirmation: true},
	{method: http.MethodGet, routePath: "/v1/user/:id/audit_logs", requestPath: "/v1/user/1/audit_logs", permission: authz.PermissionAuditReadAny, roles: []string{string(authz.StaffRoleAdmin)}},
	{method: http.MethodGet, routePath: "/v1/user/:id/roles", requestPath: "/v1/user/1/roles", permission: authz.PermissionRoleReadAny, roles: []string{string(authz.StaffRoleAdmin)}},
	{method: http.MethodPut, routePath: "/v1/user/:id/roles", requestPath: "/v1/user/1/roles", permission: authz.PermissionRoleAssignAny, roles: []string{string(authz.StaffRoleAdmin)}, needsConfirmation: true},
	{method: http.MethodPut, routePath: "/v1/user/:id/resource_scopes", requestPath: "/v1/user/1/resource_scopes", permission: authz.PermissionRoleAssignAny, roles: []string{string(authz.StaffRoleSuperAdmin)}, resourceDomain: string(authz.BusinessDomainPlatform), needsConfirmation: true},
	{method: http.MethodGet, routePath: "/v1/staff/roles", requestPath: "/v1/staff/roles", permission: authz.PermissionRoleReadAny, roles: []string{string(authz.StaffRoleAdmin)}},
	{method: http.MethodGet, routePath: "/v1/staff/sessions", requestPath: "/v1/staff/sessions", permission: authz.PermissionStaffSessionReadAny, roles: []string{string(authz.StaffRoleAdmin)}, resourceDomain: string(authz.BusinessDomainPlatform)},
	{method: http.MethodPost, routePath: "/v1/staff/sessions/:session_id/revoke", requestPath: "/v1/staff/sessions/session-1/revoke", permission: authz.PermissionStaffSessionRevokeAny, roles: []string{string(authz.StaffRoleAdmin)}, resourceDomain: string(authz.BusinessDomainPlatform), needsConfirmation: true},
	{method: http.MethodPost, routePath: "/v1/staff/:id/sessions/revoke", requestPath: "/v1/staff/1/sessions/revoke", permission: authz.PermissionStaffSessionRevokeAny, roles: []string{string(authz.StaffRoleAdmin)}, resourceDomain: string(authz.BusinessDomainPlatform), needsConfirmation: true},
	{method: http.MethodPost, routePath: "/v1/staff/roles", requestPath: "/v1/staff/roles", permission: authz.PermissionRoleWriteAny, roles: []string{string(authz.StaffRoleAdmin)}, needsConfirmation: true},
	{method: http.MethodPut, routePath: "/v1/staff/roles/:name", requestPath: "/v1/staff/roles/ops_delegate", permission: authz.PermissionRoleWriteAny, roles: []string{string(authz.StaffRoleAdmin)}, needsConfirmation: true},
	{method: http.MethodDelete, routePath: "/v1/staff/roles/:name", requestPath: "/v1/staff/roles/ops_delegate", permission: authz.PermissionRoleWriteAny, roles: []string{string(authz.StaffRoleAdmin)}, needsConfirmation: true},
	{method: http.MethodGet, routePath: "/v1/staff/permission_templates", requestPath: "/v1/staff/permission_templates", permission: authz.PermissionRoleReadAny, roles: []string{string(authz.StaffRoleAdmin)}},
	{method: http.MethodGet, routePath: "/v1/goods", requestPath: "/v1/goods", permission: authz.PermissionGoodsReadAny, roles: []string{string(authz.StaffRoleCatalog)}, resourceDomain: string(authz.BusinessDomainCatalog), resourceStoreID: "store-a"},
	{method: http.MethodGet, routePath: "/v1/goods/search/outbox", requestPath: "/v1/goods/search/outbox", permission: authz.PermissionGoodsReadAny, roles: []string{string(authz.StaffRoleCatalog)}, resourceDomain: string(authz.BusinessDomainCatalog), resourceStoreID: "store-a"},
	{method: http.MethodGet, routePath: "/v1/goods/:id", requestPath: "/v1/goods/1", permission: authz.PermissionGoodsReadAny, roles: []string{string(authz.StaffRoleCatalog)}, resourceDomain: string(authz.BusinessDomainCatalog), resourceStoreID: "store-a"},
	{method: http.MethodPost, routePath: "/v1/goods", requestPath: "/v1/goods", permission: authz.PermissionGoodsWriteAny, roles: []string{string(authz.StaffRoleCatalog)}, resourceDomain: string(authz.BusinessDomainCatalog), resourceStoreID: "store-a", needsConfirmation: true},
	{method: http.MethodPost, routePath: "/v1/goods/search/outbox/replay", requestPath: "/v1/goods/search/outbox/replay", permission: authz.PermissionGoodsWriteAny, roles: []string{string(authz.StaffRoleCatalog)}, resourceDomain: string(authz.BusinessDomainCatalog), resourceStoreID: "store-a", needsConfirmation: true},
	{method: http.MethodPost, routePath: "/v1/goods/search/reindex", requestPath: "/v1/goods/search/reindex", permission: authz.PermissionGoodsWriteAny, roles: []string{string(authz.StaffRoleCatalog)}, resourceDomain: string(authz.BusinessDomainCatalog), resourceStoreID: "store-a", needsConfirmation: true},
	{method: http.MethodPut, routePath: "/v1/goods/:id", requestPath: "/v1/goods/1", permission: authz.PermissionGoodsWriteAny, roles: []string{string(authz.StaffRoleCatalog)}, resourceDomain: string(authz.BusinessDomainCatalog), resourceStoreID: "store-a", needsConfirmation: true},
	{method: http.MethodDelete, routePath: "/v1/goods/:id", requestPath: "/v1/goods/1", permission: authz.PermissionGoodsWriteAny, roles: []string{string(authz.StaffRoleCatalog)}, resourceDomain: string(authz.BusinessDomainCatalog), resourceStoreID: "store-a", needsConfirmation: true},
	{method: http.MethodGet, routePath: "/v1/inventory/:goods_id", requestPath: "/v1/inventory/1", permission: authz.PermissionInventoryReadAny, roles: []string{string(authz.StaffRoleOps)}, resourceDomain: string(authz.BusinessDomainOps), resourceTeamID: "warehouse-a"},
	{method: http.MethodPut, routePath: "/v1/inventory/:goods_id", requestPath: "/v1/inventory/1", permission: authz.PermissionInventoryWriteAny, roles: []string{string(authz.StaffRoleOps)}, resourceDomain: string(authz.BusinessDomainOps), resourceTeamID: "warehouse-a", needsConfirmation: true},
	{method: http.MethodGet, routePath: "/v1/inventory/flows/:order_sn", requestPath: "/v1/inventory/flows/ORD-001", permission: authz.PermissionInventoryAuditReadAny, roles: []string{string(authz.StaffRoleOps)}, resourceDomain: string(authz.BusinessDomainOps), resourceTeamID: "warehouse-a"},
	{method: http.MethodGet, routePath: "/v1/inventory/:goods_id/adjustments", requestPath: "/v1/inventory/1/adjustments", permission: authz.PermissionInventoryAuditReadAny, roles: []string{string(authz.StaffRoleOps)}, resourceDomain: string(authz.BusinessDomainOps), resourceTeamID: "warehouse-a"},
	{method: http.MethodGet, routePath: "/v1/orders", requestPath: "/v1/orders", permission: authz.PermissionOrderReadAny, roles: []string{string(authz.StaffRoleOps)}, resourceDomain: string(authz.BusinessDomainOps), resourceTeamID: "warehouse-a"},
	{method: http.MethodGet, routePath: "/v1/orders/trace", requestPath: "/v1/orders/trace?order_sn=ORD-001", permission: authz.PermissionOrderReadAny, roles: []string{string(authz.StaffRoleOps)}, resourceDomain: string(authz.BusinessDomainOps), resourceTeamID: "warehouse-a"},
	{method: http.MethodGet, routePath: "/v1/orders/:order_sn", requestPath: "/v1/orders/ORD-001", permission: authz.PermissionOrderReadAny, roles: []string{string(authz.StaffRoleOps)}, resourceDomain: string(authz.BusinessDomainOps), resourceTeamID: "warehouse-a"},
	{method: http.MethodPost, routePath: "/v1/orders/:order_sn/close", requestPath: "/v1/orders/ORD-001/close", permission: authz.PermissionOrderCloseAny, roles: []string{string(authz.StaffRoleOps)}, resourceDomain: string(authz.BusinessDomainOps), resourceTeamID: "warehouse-a", needsConfirmation: true},
	{method: http.MethodPost, routePath: "/v1/orders/:order_sn/refund", requestPath: "/v1/orders/ORD-001/refund", permission: authz.PermissionOrderRefundAny, roles: []string{string(authz.StaffRoleFinance)}, resourceDomain: string(authz.BusinessDomainFinance), resourceStoreID: "store-a", needsConfirmation: true},
	{method: http.MethodGet, routePath: "/v1/payments/events", requestPath: "/v1/payments/events", permission: authz.PermissionOrderRefundAny, roles: []string{string(authz.StaffRoleFinance)}, resourceDomain: string(authz.BusinessDomainFinance), resourceStoreID: "store-a"},
	{method: http.MethodGet, routePath: "/v1/payments/reconciliation/runs", requestPath: "/v1/payments/reconciliation/runs", permission: authz.PermissionPaymentReconcileReadAny, roles: []string{string(authz.StaffRoleFinance)}, resourceDomain: string(authz.BusinessDomainFinance), resourceStoreID: "store-a"},
	{method: http.MethodGet, routePath: "/v1/payments/reconciliation/items", requestPath: "/v1/payments/reconciliation/items", permission: authz.PermissionPaymentReconcileReadAny, roles: []string{string(authz.StaffRoleFinance)}, resourceDomain: string(authz.BusinessDomainFinance), resourceStoreID: "store-a"},
	{method: http.MethodPost, routePath: "/v1/payments/refund_jobs/:id/retry", requestPath: "/v1/payments/refund_jobs/1/retry", permission: authz.PermissionRefundDeadJobRetryAny, roles: []string{string(authz.StaffRoleFinance)}, resourceDomain: string(authz.BusinessDomainFinance), resourceStoreID: "store-a", needsConfirmation: true},
	{method: http.MethodGet, routePath: "/v1/reviews", requestPath: "/v1/reviews", permission: authz.PermissionReviewModerateAny, roles: []string{string(authz.StaffRoleReview)}, resourceDomain: string(authz.BusinessDomainReview), resourceStoreID: "store-a"},
	{method: http.MethodPost, routePath: "/v1/reviews/:id/moderate", requestPath: "/v1/reviews/1/moderate", permission: authz.PermissionReviewModerateAny, roles: []string{string(authz.StaffRoleReview)}, resourceDomain: string(authz.BusinessDomainReview), resourceStoreID: "store-a"},
	{method: http.MethodPost, routePath: "/v1/reviews/:id/reply", requestPath: "/v1/reviews/1/reply", permission: authz.PermissionReviewReplyAny, roles: []string{string(authz.StaffRoleReview)}, resourceDomain: string(authz.BusinessDomainReview), resourceStoreID: "store-a"},
	{method: http.MethodPost, routePath: "/v1/reviews/ratings/:goods_id/rebuild", requestPath: "/v1/reviews/ratings/1/rebuild", permission: authz.PermissionReviewAggregateRebuild, roles: []string{string(authz.StaffRoleAdmin)}, resourceDomain: string(authz.BusinessDomainReview), resourceStoreID: "store-a", needsConfirmation: true},
}

var _ upbv1.UserClient = (*fakeAdminUserClient)(nil)
