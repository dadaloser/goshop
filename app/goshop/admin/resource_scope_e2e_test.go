package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	goodspb "goshop/api/goods/v1"
	inventorypb "goshop/api/inventory/v1"
	orderpb "goshop/api/order/v1"
	rpb "goshop/api/review/v1"
	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authz"
	"goshop/gmicro/server/restserver"
	"goshop/gmicro/server/restserver/middlewares"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestAUTH201GoodsScopeNegativeMatrix(t *testing.T) {
	server, _ := newAdminScopedBusinessServer(t)
	cfg := newAdminRBACRouteTestConfig()

	tests := []struct {
		name       string
		token      string
		wantStatus int
		wantTotal  *int
	}{
		{
			name:       "missing scope",
			token:      mustCreateScopedAdminToken(t, cfg.Jwt, 99, []string{string(authz.StaffRoleCatalog)}, []string{string(authz.PermissionGoodsReadAny)}, "", "", ""),
			wantStatus: http.StatusForbidden,
		},
		{
			name: "wrong store resource scope filters list",
			token: mustCreateScopedAdminTokenWithResourceScopes(
				t,
				cfg.Jwt,
				99,
				[]string{string(authz.StaffRoleCatalog)},
				[]string{string(authz.PermissionGoodsReadAny)},
				[]authz.ResourceScope{{Domain: string(authz.BusinessDomainCatalog), StoreID: "store-c"}},
				string(authz.BusinessDomainCatalog),
				"",
				"",
			),
			wantStatus: http.StatusOK,
			wantTotal:  intPtr(0),
		},
		{
			name: "wrong resource id filters list",
			token: mustCreateScopedAdminTokenWithResourceScopes(
				t,
				cfg.Jwt,
				99,
				[]string{string(authz.StaffRoleCatalog)},
				[]string{string(authz.PermissionGoodsReadAny)},
				[]authz.ResourceScope{{Domain: string(authz.BusinessDomainCatalog), StoreID: "store-b", ResourceType: "goods", ResourceID: "999"}},
				string(authz.BusinessDomainCatalog),
				"",
				"",
			),
			wantStatus: http.StatusOK,
			wantTotal:  intPtr(0),
		},
		{
			name:       "platform admin cannot bypass catalog domain",
			token:      mustCreateScopedAdminToken(t, cfg.Jwt, 99, []string{string(authz.StaffRoleSuperAdmin)}, []string{string(authz.PermissionGoodsReadAny)}, string(authz.BusinessDomainPlatform), "", ""),
			wantStatus: http.StatusForbidden,
		},
		{
			name: "break glass token cannot access staff route group",
			token: mustCreateAdminTokenWithClaims(t, cfg.Jwt, middlewares.CustomClaims{
				PrincipalType: string(authz.PrincipalAdminBootstrap),
				AccountStatus: string(authz.AccountStatusActive),
			}),
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/goods", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			req.Header.Set("X-Resource-Domain", string(authz.BusinessDomainCatalog))
			req.Header.Set("X-Store-ID", "store-a")
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("GET /v1/goods status = %d, want %d, body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantTotal == nil {
				return
			}
			var body struct {
				Total int                          `json:"total"`
				Data  []*goodspb.GoodsInfoResponse `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("json.Unmarshal(GET /v1/goods) error = %v", err)
			}
			if body.Total != *tt.wantTotal {
				t.Fatalf("GET /v1/goods total = %d, want %d", body.Total, *tt.wantTotal)
			}
			if len(body.Data) != *tt.wantTotal {
				t.Fatalf("GET /v1/goods items = %d, want %d", len(body.Data), *tt.wantTotal)
			}
		})
	}
}

func TestAUTH201InventoryWrongTeamScopeRejectsRead(t *testing.T) {
	server, goodsClient := newAdminScopedBusinessServer(t)
	cfg := newAdminRBACRouteTestConfig()
	token := mustCreateScopedAdminTokenWithResourceScopes(
		t,
		cfg.Jwt,
		99,
		[]string{string(authz.StaffRoleOps)},
		[]string{string(authz.PermissionInventoryReadAny)},
		[]authz.ResourceScope{{Domain: string(authz.BusinessDomainOps), TeamID: "warehouse-z"}},
		string(authz.BusinessDomainOps),
		"",
		"",
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/inventory/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Resource-Domain", string(authz.BusinessDomainOps))
	req.Header.Set("X-Team-ID", "warehouse-a")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("GET /v1/inventory/1 status = %d, want %d, body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if goodsClient.getDetailCalls == 0 {
		t.Fatalf("GET /v1/inventory/1 goods detail lookups = %d, want > 0", goodsClient.getDetailCalls)
	}
}

func TestAUTH201PlatformAdminScopeRoutesRemainAccessible(t *testing.T) {
	server, _ := newAdminScopedBusinessServer(t)
	cfg := newAdminRBACRouteTestConfig()
	token := mustCreateScopedAdminToken(
		t,
		cfg.Jwt,
		99,
		[]string{string(authz.StaffRoleAdmin)},
		[]string{string(authz.PermissionStaffSessionReadAny)},
		string(authz.BusinessDomainPlatform),
		"",
		"",
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/staff/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Resource-Domain", string(authz.BusinessDomainPlatform))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /v1/staff/sessions status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestAUTH101GoodsCrossStoreFiltersAndRejectsWrites(t *testing.T) {
	server, goodsClient := newAdminScopedBusinessServer(t)
	cfg := newAdminRBACRouteTestConfig()
	token := mustCreateScopedAdminToken(
		t,
		cfg.Jwt,
		99,
		[]string{string(authz.StaffRoleCatalog)},
		[]string{string(authz.PermissionGoodsReadAny), string(authz.PermissionGoodsWriteAny)},
		string(authz.BusinessDomainCatalog),
		"store-a",
		"",
	)

	listReq := httptest.NewRequest(http.MethodGet, "/v1/goods", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listReq.Header.Set("X-Resource-Domain", string(authz.BusinessDomainCatalog))
	listReq.Header.Set("X-Store-ID", "store-a")
	listRec := httptest.NewRecorder()
	server.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/goods status = %d, want %d, body=%s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var listBody struct {
		Total int                          `json:"total"`
		Data  []*goodspb.GoodsInfoResponse `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("json.Unmarshal(GET /v1/goods) error = %v", err)
	}
	if got := listBody.Total; got != 0 {
		t.Fatalf("GET /v1/goods total = %d, want %d", got, 0)
	}
	if got := len(listBody.Data); got != 0 {
		t.Fatalf("GET /v1/goods items = %d, want %d", got, 0)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/v1/goods/1", bytes.NewBufferString(`{"name":"updated","goodsSn":"goods-1","categoryId":1,"brandId":1,"storeId":"store-b"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("X-Resource-Domain", string(authz.BusinessDomainCatalog))
	updateReq.Header.Set("X-Store-ID", "store-a")
	updateReq.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	updateRec := httptest.NewRecorder()
	server.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusForbidden {
		t.Fatalf("PUT /v1/goods/1 status = %d, want %d, body=%s", updateRec.Code, http.StatusForbidden, updateRec.Body.String())
	}
	if got := goodsClient.updateCalls; got != 0 {
		t.Fatalf("PUT /v1/goods/1 updateCalls = %d, want %d", got, 0)
	}
}

func TestAUTH101InventoryCrossStoreRejectsReads(t *testing.T) {
	server, goodsClient := newAdminScopedBusinessServer(t)
	cfg := newAdminRBACRouteTestConfig()
	token := mustCreateScopedAdminToken(
		t,
		cfg.Jwt,
		99,
		[]string{string(authz.StaffRoleOps)},
		[]string{string(authz.PermissionInventoryReadAny)},
		string(authz.BusinessDomainOps),
		"store-a",
		"",
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/inventory/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Resource-Domain", string(authz.BusinessDomainOps))
	req.Header.Set("X-Store-ID", "store-a")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("GET /v1/inventory/1 status = %d, want %d, body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if got := goodsClient.getDetailCalls; got == 0 {
		t.Fatalf("GET /v1/inventory/1 goods detail lookups = %d, want > 0", got)
	}
}

func TestAUTH101OrdersCrossStoreFiltersAndRejectsWrites(t *testing.T) {
	server, _, orderClient, _ := newAdminScopedBusinessReviewServer(t)
	cfg := newAdminRBACRouteTestConfig()
	token := mustCreateScopedAdminToken(
		t,
		cfg.Jwt,
		99,
		[]string{string(authz.StaffRoleFinance)},
		[]string{string(authz.PermissionOrderReadAny), string(authz.PermissionOrderRefundAny)},
		string(authz.BusinessDomainFinance),
		"store-a",
		"",
	)

	listReq := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listReq.Header.Set("X-Resource-Domain", string(authz.BusinessDomainFinance))
	listReq.Header.Set("X-Store-ID", "store-a")
	listRec := httptest.NewRecorder()
	server.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/orders status = %d, want %d, body=%s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var listBody struct {
		Total int                          `json:"total"`
		Data  []*orderpb.OrderInfoResponse `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("json.Unmarshal(GET /v1/orders) error = %v", err)
	}
	if got := listBody.Total; got != 0 {
		t.Fatalf("GET /v1/orders total = %d, want %d", got, 0)
	}
	if got := len(listBody.Data); got != 0 {
		t.Fatalf("GET /v1/orders items = %d, want %d", got, 0)
	}

	refundReq := httptest.NewRequest(http.MethodPost, "/v1/orders/ORD-001/refund", bytes.NewBufferString(`{"amount_fen":100,"reason":"manual refund"}`))
	refundReq.Header.Set("Content-Type", "application/json")
	refundReq.Header.Set("Authorization", "Bearer "+token)
	refundReq.Header.Set("X-Resource-Domain", string(authz.BusinessDomainFinance))
	refundReq.Header.Set("X-Store-ID", "store-a")
	refundReq.Header.Set("X-Admin-Confirm-Token", "confirm-secret")
	refundRec := httptest.NewRecorder()
	server.ServeHTTP(refundRec, refundReq)
	if refundRec.Code != http.StatusForbidden {
		t.Fatalf("POST /v1/orders/ORD-001/refund status = %d, want %d, body=%s", refundRec.Code, http.StatusForbidden, refundRec.Body.String())
	}
	if got := orderClient.updateStatusCalls; got != 0 {
		t.Fatalf("POST /v1/orders/ORD-001/refund updateStatusCalls = %d, want %d", got, 0)
	}
}

func TestAUTH101ReviewsCrossStoreFiltersAndRejectsWrites(t *testing.T) {
	server, _, _, reviewClient := newAdminScopedBusinessReviewServer(t)
	cfg := newAdminRBACRouteTestConfig()
	token := mustCreateScopedAdminToken(
		t,
		cfg.Jwt,
		99,
		[]string{string(authz.StaffRoleReview)},
		[]string{string(authz.PermissionReviewModerateAny), string(authz.PermissionReviewReplyAny)},
		string(authz.BusinessDomainReview),
		"store-a",
		"",
	)

	listReq := httptest.NewRequest(http.MethodGet, "/v1/reviews", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listReq.Header.Set("X-Resource-Domain", string(authz.BusinessDomainReview))
	listReq.Header.Set("X-Store-ID", "store-a")
	listRec := httptest.NewRecorder()
	server.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/reviews status = %d, want %d, body=%s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var listBody struct {
		Total int                   `json:"total"`
		Data  []*rpb.ReviewResponse `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("json.Unmarshal(GET /v1/reviews) error = %v", err)
	}
	if got := listBody.Total; got != 0 {
		t.Fatalf("GET /v1/reviews total = %d, want %d", got, 0)
	}
	if got := len(listBody.Data); got != 0 {
		t.Fatalf("GET /v1/reviews items = %d, want %d", got, 0)
	}

	moderateReq := httptest.NewRequest(http.MethodPost, "/v1/reviews/1/moderate", bytes.NewBufferString(`{"decision":"APPROVED","reason":"cross-store should fail"}`))
	moderateReq.Header.Set("Content-Type", "application/json")
	moderateReq.Header.Set("Authorization", "Bearer "+token)
	moderateReq.Header.Set("X-Resource-Domain", string(authz.BusinessDomainReview))
	moderateReq.Header.Set("X-Store-ID", "store-a")
	moderateRec := httptest.NewRecorder()
	server.ServeHTTP(moderateRec, moderateReq)
	if moderateRec.Code != http.StatusForbidden {
		t.Fatalf("POST /v1/reviews/1/moderate status = %d, want %d, body=%s", moderateRec.Code, http.StatusForbidden, moderateRec.Body.String())
	}
	if got := reviewClient.moderateCalls; got != 0 {
		t.Fatalf("POST /v1/reviews/1/moderate moderateCalls = %d, want %d", got, 0)
	}
}

func newAdminScopedBusinessServer(t *testing.T) (*restserver.Server, *scopeGoodsClient) {
	t.Helper()
	server, goodsClient, _, _ := newAdminScopedBusinessReviewServer(t)
	return server, goodsClient
}

func intPtr(value int) *int { return &value }

func newAdminScopedBusinessReviewServer(t *testing.T) (*restserver.Server, *scopeGoodsClient, *scopeOrderClient, *scopeReviewClient) {
	t.Helper()

	server := restserver.NewServer()
	cfg := newAdminRBACRouteTestConfig()
	userClient := &fakeAdminUserClient{
		authUserResponse: &upbv1.UserAuthResponse{
			User: &upbv1.UserInfoResponse{
				Id:       99,
				Username: "scope_admin",
				Status:   string(authz.AccountStatusActive),
			},
			LegacyRole: int32(authz.LegacyUserRoleAdmin),
		},
		staffRolesResponse: &upbv1.StaffRoleListResponse{},
	}

	goodsClient := &scopeGoodsClient{
		listResponse: &goodspb.GoodsListResponse{
			Total: 1,
			Data: []*goodspb.GoodsInfoResponse{
				{Id: 1, Name: "cross-store-goods", StoreId: "store-b"},
			},
		},
		detailByID: map[int32]*goodspb.GoodsInfoResponse{
			1:   {Id: 1, Name: "cross-store-goods", StoreId: "store-b"},
			101: {Id: 101, Name: "cross-store-review-goods", StoreId: "store-b"},
			201: {Id: 201, Name: "cross-store-other-goods", StoreId: "store-b"},
		},
	}
	orderClient := &scopeOrderClient{
		listResponse: &orderpb.OrderListResponse{
			Total: 1,
			Data: []*orderpb.OrderInfoResponse{
				{Id: 1, OrderSn: "ORD-001", StoreId: "store-b", Status: "TRADE_SUCCESS"},
			},
		},
		detailBySN: map[string]*orderpb.OrderInfoDetailResponse{
			"ORD-001": {
				OrderInfo: &orderpb.OrderInfoResponse{Id: 1, OrderSn: "ORD-001", StoreId: "store-b", Status: "TRADE_SUCCESS"},
			},
		},
	}
	inventoryClient := &scopeInventoryClient{}
	reviewClient := &scopeReviewClient{
		listResponse: &rpb.ReviewListResponse{
			Total: 1,
			Data: []*rpb.ReviewResponse{
				{Id: 1, GoodsId: 101, Status: "PENDING", Content: "cross-store review"},
			},
		},
		reviewByID: map[int64]*rpb.ReviewResponse{
			1: {Id: 1, GoodsId: 101, Status: "PENDING", Content: "cross-store review"},
		},
	}

	if err := initRouterWithDependencies(server, cfg, userClient, goodsClient, inventoryClient, orderClient, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("initRouterWithDependencies() error = %v", err)
	}
	if err := registerAdminReviewRoutesWithStores(server, cfg, userClient, goodsClient, reviewClient, &fakeAdminRevocationStore{}, &fakeAdminTokenVersionStore{}); err != nil {
		t.Fatalf("registerAdminReviewRoutesWithStores() error = %v", err)
	}
	return server, goodsClient, orderClient, reviewClient
}

type scopeGoodsClient struct {
	goodspb.GoodsClient
	listResponse   *goodspb.GoodsListResponse
	detailByID     map[int32]*goodspb.GoodsInfoResponse
	getDetailCalls int
	updateCalls    int
}

func (f *scopeGoodsClient) GoodsList(context.Context, *goodspb.GoodsFilterRequest, ...grpc.CallOption) (*goodspb.GoodsListResponse, error) {
	return f.listResponse, nil
}

func (f *scopeGoodsClient) GetGoodsDetail(_ context.Context, req *goodspb.GoodInfoRequest, _ ...grpc.CallOption) (*goodspb.GoodsInfoResponse, error) {
	f.getDetailCalls++
	if item, ok := f.detailByID[req.GetId()]; ok {
		return item, nil
	}
	return &goodspb.GoodsInfoResponse{Id: req.GetId(), StoreId: "store-b"}, nil
}

func (f *scopeGoodsClient) UpdateGoods(context.Context, *goodspb.CreateGoodsInfo, ...grpc.CallOption) (*emptypb.Empty, error) {
	f.updateCalls++
	return &emptypb.Empty{}, nil
}

type scopeInventoryClient struct {
	inventorypb.InventoryClient
	getStockCalls int
}

func (f *scopeInventoryClient) GetStock(context.Context, *inventorypb.GoodsInvInfo, ...grpc.CallOption) (*inventorypb.GoodsInvInfo, error) {
	f.getStockCalls++
	return &inventorypb.GoodsInvInfo{}, nil
}

type scopeOrderClient struct {
	orderpb.OrderClient
	listResponse      *orderpb.OrderListResponse
	detailBySN        map[string]*orderpb.OrderInfoDetailResponse
	updateStatusCalls int
}

func (f *scopeOrderClient) OrderList(context.Context, *orderpb.OrderFilterRequest, ...grpc.CallOption) (*orderpb.OrderListResponse, error) {
	return f.listResponse, nil
}

func (f *scopeOrderClient) GetOrderBySn(_ context.Context, req *orderpb.OrderLookupRequest, _ ...grpc.CallOption) (*orderpb.OrderInfoDetailResponse, error) {
	if item, ok := f.detailBySN[req.GetOrderSn()]; ok {
		return item, nil
	}
	return &orderpb.OrderInfoDetailResponse{
		OrderInfo: &orderpb.OrderInfoResponse{OrderSn: req.GetOrderSn(), StoreId: "store-b", Status: "TRADE_SUCCESS"},
	}, nil
}

func (f *scopeOrderClient) UpdateOrderStatus(context.Context, *orderpb.OrderStatus, ...grpc.CallOption) (*emptypb.Empty, error) {
	f.updateStatusCalls++
	return &emptypb.Empty{}, nil
}

type scopeReviewClient struct {
	rpb.ReviewClient
	listResponse  *rpb.ReviewListResponse
	reviewByID    map[int64]*rpb.ReviewResponse
	moderateCalls int
}

func (f *scopeReviewClient) ListReviews(context.Context, *rpb.ListReviewsRequest, ...grpc.CallOption) (*rpb.ReviewListResponse, error) {
	return f.listResponse, nil
}

func (f *scopeReviewClient) GetReview(_ context.Context, req *rpb.GetReviewRequest, _ ...grpc.CallOption) (*rpb.ReviewResponse, error) {
	if item, ok := f.reviewByID[req.GetReviewId()]; ok {
		return item, nil
	}
	return &rpb.ReviewResponse{Id: req.GetReviewId(), GoodsId: 101, Status: "PENDING"}, nil
}

func (f *scopeReviewClient) ModerateReview(context.Context, *rpb.ModerateReviewRequest, ...grpc.CallOption) (*rpb.ReviewResponse, error) {
	f.moderateCalls++
	return &rpb.ReviewResponse{}, nil
}
