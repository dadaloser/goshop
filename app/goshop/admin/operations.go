package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	goodspb "goshop/api/goods/v1"
	inventorypb "goshop/api/inventory/v1"
	orderpb "goshop/api/order/v1"
	userpb "goshop/api/user/v1"
	"goshop/app/goshop/admin/config"
	"goshop/app/pkg/authz"
	"goshop/gmicro/server/restserver/middlewares"
	gauth "goshop/gmicro/server/restserver/middlewares/auth"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"
	core "goshop/pkg/transport/httperror"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const operationCorrelationKey = "ADMIN_OPERATION_CORRELATION_ID"

type operationsHandler struct {
	users     userpb.UserClient
	goods     goodspb.GoodsClient
	inventory inventorypb.InventoryClient
	orders    orderpb.OrderClient
}

func newOperationsHandler(users userpb.UserClient, goods goodspb.GoodsClient, inventory inventorypb.InventoryClient, orders orderpb.OrderClient) *operationsHandler {
	return &operationsHandler{users: users, goods: goods, inventory: inventory, orders: orders}
}

func registerOperationsRoutes(v1 *gin.RouterGroup, staffAuth middlewares.AuthStrategy, cfg *config.Config, h *operationsHandler) {
	staff := []gin.HandlerFunc{staffAuth.AuthFunc(), authz.RequirePrincipalTypes(authz.PrincipalStaff)}
	goods := v1.Group("/goods", staff...)
	goods.Use(requireRole(authz.StaffRoleCatalog, authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireResourceScope(authz.BusinessDomainCatalog))
	goods.GET("", authz.RequirePermission(authz.PermissionGoodsReadAny), h.listGoods)
	goods.GET("search/outbox", authz.RequirePermission(authz.PermissionGoodsReadAny), h.listGoodsOutboxEvents)
	goods.GET(":id", authz.RequirePermission(authz.PermissionGoodsReadAny), h.getGoods)
	goods.POST("", authz.RequirePermission(authz.PermissionGoodsWriteAny), requireAdminConfirmation(cfg.AdminAuth), h.createGoods)
	goods.POST("search/outbox/replay", authz.RequirePermission(authz.PermissionGoodsWriteAny), requireAdminConfirmation(cfg.AdminAuth), h.replayGoodsOutbox)
	goods.POST("search/reindex", authz.RequirePermission(authz.PermissionGoodsWriteAny), requireAdminConfirmation(cfg.AdminAuth), h.reindexGoods)
	goods.PUT(":id", authz.RequirePermission(authz.PermissionGoodsWriteAny), requireAdminConfirmation(cfg.AdminAuth), h.updateGoods)
	goods.DELETE(":id", authz.RequirePermission(authz.PermissionGoodsWriteAny), requireAdminConfirmation(cfg.AdminAuth), h.deleteGoods)

	inventory := v1.Group("/inventory", staff...)
	inventory.Use(requireRole(authz.StaffRoleOps, authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireResourceScope(authz.BusinessDomainOps))
	inventory.GET(":goods_id", authz.RequirePermission(authz.PermissionInventoryReadAny), h.getInventory)
	inventory.PUT(":goods_id", authz.RequirePermission(authz.PermissionInventoryWriteAny), requireAdminConfirmation(cfg.AdminAuth), h.adjustInventory)
	inventory.GET("flows/:order_sn", authz.RequirePermission(authz.PermissionInventoryAuditReadAny), h.inventoryFlow)
	inventory.GET(":goods_id/adjustments", authz.RequirePermission(authz.PermissionInventoryAuditReadAny), h.inventoryAdjustments)

	orders := v1.Group("/orders", staff...)
	orders.GET("", requireResourceScopeForRoles(), authz.RequirePermission(authz.PermissionOrderReadAny), h.listOrders)
	orders.GET("trace", requireResourceScopeForRoles(), authz.RequirePermission(authz.PermissionOrderReadAny), h.getOrderTrace)
	orders.GET(":order_sn", requireResourceScopeForRoles(), authz.RequirePermission(authz.PermissionOrderReadAny), h.getOrder)
	orders.POST(":order_sn/close", requireRole(authz.StaffRoleOps, authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), authz.RequirePermission(authz.PermissionOrderCloseAny), requireAdminConfirmation(cfg.AdminAuth), h.closeOrder)
	orders.POST(":order_sn/refund", requireRole(authz.StaffRoleFinance, authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), authz.RequirePermission(authz.PermissionOrderRefundAny), requireAdminConfirmation(cfg.AdminAuth), h.refundOrder)
	payments := v1.Group("/payments", staff...)
	payments.Use(requireRole(authz.StaffRoleFinance, authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireResourceScope(authz.BusinessDomainFinance))
	payments.GET("events", authz.RequirePermission(authz.PermissionOrderRefundAny), h.listPaymentEvents)
	payments.GET("reconciliation/runs", authz.RequirePermission(authz.PermissionPaymentReconcileReadAny), h.listPaymentReconciliationRuns)
	payments.GET("reconciliation/items", authz.RequirePermission(authz.PermissionPaymentReconcileReadAny), h.listPaymentReconciliationItems)
	payments.POST("refund_jobs/:id/retry", authz.RequirePermission(authz.PermissionRefundDeadJobRetryAny), requireAdminConfirmation(cfg.AdminAuth), h.retryDeadRefundJob)
}

func requireRole(allowed ...authz.StaffRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := stringSet(gauth.ExtractClaims(c)["roles"])
		for _, role := range allowed {
			if roles[string(role)] {
				c.Next()
				return
			}
		}
		core.AbortWithError(c, apperrors.NewCode(errcode.ErrPermissionDenied, "role is not allowed for this operation"))
	}
}

func requireResourceScope(domain authz.BusinessDomain) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !claimsAllowDomain(gauth.ExtractClaims(c), domain) {
			core.AbortWithError(c, apperrors.NewCode(errcode.ErrPermissionDenied, "resource scope denied"))
			return
		}
		c.Next()
	}
}

func requireTargetResourceScope(domain authz.BusinessDomain, resourceType, param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		resourceID := strings.TrimSpace(c.Param(param))
		if resourceID == "" {
			core.AbortWithError(c, apperrors.NewCode(errcode.ErrValidation, "resource id is required"))
			return
		}
		requireResourceScope(domain)(c)
		if c.IsAborted() {
			return
		}
		c.Next()
	}
}

func requireResourceScopeForRoles() gin.HandlerFunc {
	return func(c *gin.Context) {
		roles := stringSet(gauth.ExtractClaims(c)["roles"])
		domain := authz.BusinessDomainSupport
		if roles[string(authz.StaffRoleFinance)] {
			domain = authz.BusinessDomainFinance
		}
		if roles[string(authz.StaffRoleOps)] {
			domain = authz.BusinessDomainOps
		}
		requireResourceScope(domain)(c)
	}
}

func scopeAllows(raw any, requested string) bool {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return true
	}
	values := stringSet(raw)
	return values["*"] || values[requested]
}

func claimsAllowScope(claims map[string]any, requested authz.ResourceScope) bool {
	resourceScopes := authz.ParseResourceScopes(claims["resource_scopes"])
	if len(resourceScopes) > 0 {
		return authz.ResourceScopeAllows(resourceScopes, requested)
	}
	if !scopeAllows(claims["resource_domains"], requested.Domain) {
		return false
	}
	if !scopeAllows(claims["resource_stores"], requested.StoreID) {
		return false
	}
	if !scopeAllows(claims["resource_teams"], requested.TeamID) {
		return false
	}
	return true
}

func claimsAllowDomain(claims map[string]any, domain authz.BusinessDomain) bool {
	resourceScopes := authz.ParseResourceScopes(claims["resource_scopes"])
	for _, scope := range resourceScopes {
		if strings.EqualFold(scope.Domain, string(domain)) {
			return true
		}
	}
	return scopeAllows(claims["resource_domains"], string(domain))
}

func stringSet(raw any) map[string]bool {
	result := map[string]bool{}
	switch values := raw.(type) {
	case []string:
		for _, value := range values {
			result[value] = true
		}
	case []any:
		for _, value := range values {
			if text, ok := value.(string); ok {
				result[text] = true
			}
		}
	}
	return result
}

func (h *operationsHandler) listGoods(c *gin.Context) {
	if !h.ready(c, h.goods != nil) {
		return
	}
	page, size := page(c)
	resp, err := h.goods.GoodsList(c, &goodspb.GoodsFilterRequest{Pages: page, PagePerNums: size, KeyWords: c.Query("keywords"), SpuCode: strings.TrimSpace(c.Query("spu_code")), SkuCode: strings.TrimSpace(c.Query("sku_code")), IncludeOffSale: true})
	if err == nil && resp != nil {
		filtered := make([]*goodspb.GoodsInfoResponse, 0, len(resp.GetData()))
		for _, item := range resp.GetData() {
			if claimsAllowScope(gauth.ExtractClaims(c), goodsScope(item, authz.BusinessDomainCatalog)) {
				filtered = append(filtered, item)
			}
		}
		resp.Data = filtered
		resp.Total = int32(len(filtered))
	}
	writeRPC(c, resp, err)
}
func (h *operationsHandler) getGoods(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	resp, ok := h.authorizeGoods(c, id, authz.BusinessDomainCatalog)
	if !ok {
		return
	}
	writeRPC(c, resp, nil)
}
func (h *operationsHandler) createGoods(c *gin.Context) {
	var req goodspb.CreateGoodsInfo
	if !h.bind(c, h.goods != nil, &req) {
		return
	}
	if !claimsAllowScope(gauth.ExtractClaims(c), authz.ResourceScope{Domain: string(authz.BusinessDomainCatalog), StoreID: strings.TrimSpace(req.GetStoreId())}) {
		denyScope(c)
		return
	}
	resp, err := h.goods.CreateGoods(c, &req)
	if err == nil {
		err = h.audit(c, "goods_created", "goods", strconv.Itoa(int(resp.GetId())))
	}
	writeRPC(c, resp, err)
}
func (h *operationsHandler) updateGoods(c *gin.Context) {
	var req goodspb.CreateGoodsInfo
	if !h.bind(c, h.goods != nil, &req) {
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if _, ok = h.authorizeGoods(c, id, authz.BusinessDomainCatalog); !ok {
		return
	}
	req.Id = id
	resp, err := h.goods.UpdateGoods(c, &req)
	if err == nil {
		err = h.audit(c, "goods_updated", "goods", strconv.Itoa(int(id)))
	}
	writeRPC(c, resp, err)
}
func (h *operationsHandler) deleteGoods(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if _, ok = h.authorizeGoods(c, id, authz.BusinessDomainCatalog); !ok {
		return
	}
	resp, err := h.goods.DeleteGoods(c, &goodspb.DeleteGoodsInfo{Id: id})
	if err == nil {
		err = h.audit(c, "goods_deleted", "goods", strconv.Itoa(int(id)))
	}
	writeRPC(c, resp, err)
}

func (h *operationsHandler) listGoodsOutboxEvents(c *gin.Context) {
	if !h.ready(c, h.goods != nil) {
		return
	}
	p, s := page(c)
	resp, err := h.goods.ListGoodsOutboxEvents(c, &goodspb.ListGoodsOutboxEventsRequest{
		Topic:    strings.TrimSpace(c.Query("topic")),
		Status:   strings.TrimSpace(c.Query("status")),
		Page:     p,
		PageSize: s,
	})
	writeRPC(c, resp, err)
}

func (h *operationsHandler) replayGoodsOutbox(c *gin.Context) {
	if !h.ready(c, h.goods != nil) {
		return
	}
	var body struct {
		IDs    []int32 `json:"ids"`
		Status string  `json:"status"`
		Limit  int32   `json:"limit"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid request"))
		return
	}
	resp, err := h.goods.ReplayGoodsOutbox(c, &goodspb.ListGoodsOutboxReplayRequest{
		Ids:    body.IDs,
		Status: strings.TrimSpace(body.Status),
		Limit:  body.Limit,
	})
	if err == nil {
		err = h.audit(c, "goods_outbox_replayed", "goods_outbox", fmt.Sprintf("%v", resp.GetIds()))
	}
	writeRPC(c, resp, err)
}

func (h *operationsHandler) reindexGoods(c *gin.Context) {
	if !h.ready(c, h.goods != nil) {
		return
	}
	var body struct {
		GoodsIDs []int32 `json:"goods_ids"`
		All      bool    `json:"all"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid request"))
		return
	}
	if len(body.GoodsIDs) > 0 {
		for _, id := range body.GoodsIDs {
			if _, ok := h.authorizeGoods(c, id, authz.BusinessDomainCatalog); !ok {
				return
			}
		}
	}
	resp, err := h.goods.ReindexGoods(c, &goodspb.GoodsReindexRequest{GoodsIds: body.GoodsIDs, All: body.All})
	if err == nil {
		target := "all"
		if !body.All {
			target = fmt.Sprintf("%v", resp.GetGoodsIds())
		}
		err = h.audit(c, "goods_reindexed", "goods", target)
	}
	writeRPC(c, resp, err)
}

func (h *operationsHandler) getInventory(c *gin.Context) {
	id, ok := pathID(c, "goods_id")
	if !ok {
		return
	}
	if _, ok = h.authorizeGoods(c, id, authz.BusinessDomainOps); !ok {
		return
	}
	resp, err := h.inventory.GetStock(c, &inventorypb.GoodsInvInfo{GoodsId: id})
	writeRPC(c, resp, err)
}
func (h *operationsHandler) adjustInventory(c *gin.Context) {
	var req inventorypb.GoodsInvInfo
	if !h.bind(c, h.inventory != nil, &req) {
		return
	}
	id, ok := pathID(c, "goods_id")
	if !ok {
		return
	}
	if _, ok = h.authorizeGoods(c, id, authz.BusinessDomainOps); !ok {
		return
	}
	req.GoodsId = id
	actor, err := userIDFromClaims(c)
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	correlation, _ := c.Get(operationCorrelationKey)
	req.ActorUserId = int32(actor)
	req.CorrelationId = fmt.Sprint(correlation)
	req.RequestId = requestID(c)
	if strings.TrimSpace(req.Reason) == "" {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "adjustment reason is required"))
		return
	}
	resp, err := h.inventory.SetStock(c, &req)
	if err == nil {
		err = h.audit(c, "inventory_adjusted", "goods", strconv.Itoa(int(id)))
	}
	writeRPC(c, resp, err)
}
func (h *operationsHandler) inventoryAdjustments(c *gin.Context) {
	id, ok := pathID(c, "goods_id")
	if !ok {
		return
	}
	if _, ok = h.authorizeGoods(c, id, authz.BusinessDomainOps); !ok {
		return
	}
	p, s := page(c)
	resp, err := h.inventory.ListAdjustments(c, &inventorypb.InventoryAdjustmentListRequest{GoodsId: id, Page: p, PageSize: s})
	writeRPC(c, resp, err)
}
func (h *operationsHandler) inventoryFlow(c *gin.Context) {
	if !h.ready(c, h.inventory != nil) {
		return
	}
	if _, ok := h.authorizeOrder(c, strings.TrimSpace(c.Param("order_sn")), authz.BusinessDomainOps); !ok {
		return
	}
	resp, err := h.inventory.GetSellDetail(c, &inventorypb.OrderInfo{OrderSn: strings.TrimSpace(c.Param("order_sn"))})
	writeRPC(c, resp, err)
}

func (h *operationsHandler) listOrders(c *gin.Context) {
	if !h.ready(c, h.orders != nil) {
		return
	}
	page, size := page(c)
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 32)
	resp, err := h.orders.OrderList(c, &orderpb.OrderFilterRequest{UserId: int32(userID), Pages: page, PagePerNums: size})
	if err == nil && resp != nil {
		domain := requestedOrderDomain(c)
		filtered := make([]*orderpb.OrderInfoResponse, 0, len(resp.GetData()))
		for _, item := range resp.GetData() {
			if claimsAllowScope(gauth.ExtractClaims(c), authz.ResourceScope{Domain: string(domain), StoreID: strings.TrimSpace(item.GetStoreId()), ResourceType: "order", ResourceID: strings.TrimSpace(item.GetOrderSn())}) {
				filtered = append(filtered, item)
			}
		}
		resp.Data = filtered
		resp.Total = int32(len(filtered))
	}
	writeRPC(c, resp, err)
}
func (h *operationsHandler) getOrder(c *gin.Context) {
	resp, ok := h.authorizeOrder(c, strings.TrimSpace(c.Param("order_sn")), requestedOrderDomain(c))
	if !ok {
		return
	}
	writeRPC(c, resp, nil)
}
func (h *operationsHandler) getOrderTrace(c *gin.Context) {
	if !h.ready(c, h.orders != nil) {
		return
	}
	resp, err := h.orders.GetOrderTrace(c, &orderpb.OrderTraceRequest{
		OrderSn:       strings.TrimSpace(c.Query("order_sn")),
		TradeNo:       strings.TrimSpace(c.Query("trade_no")),
		CorrelationId: strings.TrimSpace(c.Query("correlation_id")),
	})
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	if !claimsAllowScope(gauth.ExtractClaims(c), orderScope(resp.GetOrder(), requestedOrderDomain(c))) {
		denyScope(c)
		return
	}
	writeRPC(c, resp, nil)
}
func (h *operationsHandler) closeOrder(c *gin.Context) {
	h.changeOrderStatus(c, "TRADE_CLOSED", "order_closed")
}
func (h *operationsHandler) refundOrder(c *gin.Context) {
	if !h.ready(c, h.orders != nil) {
		return
	}
	var body struct {
		AmountFen int64  `json:"amount_fen" binding:"required,gt=0"`
		Reason    string `json:"reason" binding:"required,max=255"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid refund request"))
		return
	}
	actor, err := userIDFromClaims(c)
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	correlation, _ := c.Get(operationCorrelationKey)
	orderSN := strings.TrimSpace(c.Param("order_sn"))
	if _, ok := h.authorizeOrder(c, orderSN, authz.BusinessDomainFinance); !ok {
		return
	}
	resp, err := h.orders.UpdateOrderStatus(c, &orderpb.OrderStatus{OrderSn: orderSN, Status: "REFUND_PENDING", ActorUserId: int32(actor), RefundAmountFen: body.AmountFen, Reason: strings.TrimSpace(body.Reason), CorrelationId: fmt.Sprint(correlation), RequestId: requestID(c)})
	if err == nil {
		err = h.audit(c, "order_refund_requested", "order", orderSN)
	}
	writeRPC(c, resp, err)
}
func (h *operationsHandler) changeOrderStatus(c *gin.Context, status, action string) {
	if !h.ready(c, h.orders != nil) {
		return
	}
	orderSN := strings.TrimSpace(c.Param("order_sn"))
	if _, ok := h.authorizeOrder(c, orderSN, authz.BusinessDomainOps); !ok {
		return
	}
	resp, err := h.orders.UpdateOrderStatus(c, &orderpb.OrderStatus{OrderSn: orderSN, Status: status})
	if err == nil {
		err = h.audit(c, action, "order", orderSN)
	}
	writeRPC(c, resp, err)
}

func (h *operationsHandler) listPaymentEvents(c *gin.Context) {
	if !h.ready(c, h.orders != nil) {
		return
	}
	p, s := page(c)
	resp, err := h.orders.ListPaymentEvents(c, &orderpb.PaymentEventListRequest{OrderSn: strings.TrimSpace(c.Query("order_sn")), Page: p, PageSize: s})
	writeRPC(c, resp, err)
}

func (h *operationsHandler) listPaymentReconciliationRuns(c *gin.Context) {
	if !h.ready(c, h.orders != nil) {
		return
	}
	p, s := page(c)
	resp, err := h.orders.ListPaymentReconciliationRuns(c, &orderpb.ListPaymentReconciliationRunsRequest{
		Provider:    strings.TrimSpace(c.Query("provider")),
		WindowStart: parseUnixQuery(c.Query("window_start")),
		WindowEnd:   parseUnixQuery(c.Query("window_end")),
		Page:        p,
		PageSize:    s,
	})
	writeRPC(c, resp, err)
}

func (h *operationsHandler) listPaymentReconciliationItems(c *gin.Context) {
	if !h.ready(c, h.orders != nil) {
		return
	}
	p, s := page(c)
	runID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("run_id")), 10, 64)
	resp, err := h.orders.ListPaymentReconciliationItems(c, &orderpb.ListPaymentReconciliationItemsRequest{
		Provider:    strings.TrimSpace(c.Query("provider")),
		WindowStart: parseUnixQuery(c.Query("window_start")),
		WindowEnd:   parseUnixQuery(c.Query("window_end")),
		Result:      strings.TrimSpace(c.Query("result")),
		RunId:       runID,
		Page:        p,
		PageSize:    s,
	})
	writeRPC(c, resp, err)
}

func (h *operationsHandler) retryDeadRefundJob(c *gin.Context) {
	if !h.ready(c, h.orders != nil) {
		return
	}
	id, ok := pathInt64(c, "id")
	if !ok {
		return
	}
	resp, err := h.orders.RetryDeadRefundJob(c, &orderpb.RetryDeadRefundJobRequest{Id: id})
	if err == nil {
		err = h.audit(c, "refund_dead_job_retried", "refund_job", strconv.FormatInt(id, 10))
	}
	writeRPC(c, resp, err)
}

func (h *operationsHandler) ready(c *gin.Context, ready bool) bool {
	if ready {
		return true
	}
	core.WriteError(c, apperrors.NewCode(errcode.ErrServiceUnavailable, "business rpc client is not initialized"))
	return false
}
func (h *operationsHandler) bind(c *gin.Context, ready bool, dst any) bool {
	if !h.ready(c, ready) {
		return false
	}
	if err := c.ShouldBindJSON(dst); err != nil {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid request"))
		return false
	}
	return true
}
func (h *operationsHandler) audit(c *gin.Context, action, targetType, targetID string) error {
	if h.users == nil {
		return fmt.Errorf("audit rpc client is not initialized")
	}
	actor, err := userIDFromClaims(c)
	if err != nil {
		return err
	}
	correlationID, _ := c.Get(operationCorrelationKey)
	if correlationID == nil {
		correlationID = uuid.NewString()
	}
	requestID := requestID(c)
	_, err = h.users.CreateAdminAuditLog(c, &userpb.CreateAdminAuditLogRequest{Log: &userpb.AdminAuditLog{ActorUserId: int32(actor), ActorPrincipalType: string(authz.PrincipalStaff), Action: action, Detail: fmt.Sprintf("target_type:%s target_id:%s", targetType, targetID), CorrelationId: fmt.Sprint(correlationID), RequestId: requestID, TargetType: targetType, TargetId: targetID, Domain: c.GetHeader("X-Resource-Domain"), StoreId: c.GetHeader("X-Store-ID"), TeamId: c.GetHeader("X-Team-ID")}})
	return err
}
func requestID(c *gin.Context) string {
	id := strings.TrimSpace(c.GetHeader("X-Request-ID"))
	if id == "" {
		id = uuid.NewString()
	}
	return id
}

func pathID(c *gin.Context, name string) (int32, bool) {
	value, err := strconv.ParseInt(c.Param(name), 10, 32)
	if err != nil || value <= 0 {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid resource id"))
		return 0, false
	}
	return int32(value), true
}

func pathInt64(c *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || value <= 0 {
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid resource id"))
		return 0, false
	}
	return value, true
}

func parseUnixQuery(value string) int64 {
	parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if parsed < 0 {
		return 0
	}
	return parsed
}
func page(c *gin.Context) (int32, int32) {
	p, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 32)
	s, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 32)
	if p < 1 {
		p = 1
	}
	if s < 1 || s > 100 {
		s = 20
	}
	return int32(p), int32(s)
}
func writeRPC(c *gin.Context, response any, err error) {
	if err != nil {
		core.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}
