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
	goods.GET(":id", authz.RequirePermission(authz.PermissionGoodsReadAny), h.getGoods)
	goods.POST("", authz.RequirePermission(authz.PermissionGoodsWriteAny), requireAdminConfirmation(cfg.AdminAuth), h.createGoods)
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
	orders.GET(":order_sn", requireResourceScopeForRoles(), authz.RequirePermission(authz.PermissionOrderReadAny), h.getOrder)
	orders.POST(":order_sn/close", requireRole(authz.StaffRoleOps, authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireResourceScope(authz.BusinessDomainOps), authz.RequirePermission(authz.PermissionOrderCloseAny), requireAdminConfirmation(cfg.AdminAuth), h.closeOrder)
	orders.POST(":order_sn/refund", requireRole(authz.StaffRoleFinance, authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireResourceScope(authz.BusinessDomainFinance), authz.RequirePermission(authz.PermissionOrderRefundAny), requireAdminConfirmation(cfg.AdminAuth), h.refundOrder)
	payments := v1.Group("/payments", staff...)
	payments.Use(requireRole(authz.StaffRoleFinance, authz.StaffRoleAdmin, authz.StaffRoleSuperAdmin), requireResourceScope(authz.BusinessDomainFinance), authz.RequirePermission(authz.PermissionOrderRefundAny))
	payments.GET("events", h.listPaymentEvents)
	payments.GET("reconciliation", h.reconcilePayments)
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
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "msg": "role is not allowed for this operation"})
	}
}

func requireResourceScope(domain authz.BusinessDomain) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(c.GetHeader("X-Resource-Domain")) != string(domain) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "msg": "resource domain denied"})
			return
		}
		storeID := strings.TrimSpace(c.GetHeader("X-Store-ID"))
		teamID := strings.TrimSpace(c.GetHeader("X-Team-ID"))
		if !authz.ResourceScopeMatchesDomain(domain, storeID, teamID) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "msg": "resource scope shape denied"})
			return
		}
		claims := gauth.ExtractClaims(c)
		if !scopeAllows(claims["resource_domains"], string(domain)) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "msg": "resource domain denied"})
			return
		}
		if !scopeAllows(claims["resource_stores"], storeID) || !scopeAllows(claims["resource_teams"], teamID) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "msg": "resource scope denied"})
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
	writeRPC(c, resp, err)
}
func (h *operationsHandler) getGoods(c *gin.Context) {
	if !h.ready(c, h.goods != nil) {
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.goods.GetGoodsDetail(c, &goodspb.GoodInfoRequest{Id: id})
	writeRPC(c, resp, err)
}
func (h *operationsHandler) createGoods(c *gin.Context) {
	var req goodspb.CreateGoodsInfo
	if !h.bind(c, h.goods != nil, &req) {
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
	req.Id = id
	resp, err := h.goods.UpdateGoods(c, &req)
	if err == nil {
		err = h.audit(c, "goods_updated", "goods", strconv.Itoa(int(id)))
	}
	writeRPC(c, resp, err)
}
func (h *operationsHandler) deleteGoods(c *gin.Context) {
	if !h.ready(c, h.goods != nil) {
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.goods.DeleteGoods(c, &goodspb.DeleteGoodsInfo{Id: id})
	if err == nil {
		err = h.audit(c, "goods_deleted", "goods", strconv.Itoa(int(id)))
	}
	writeRPC(c, resp, err)
}

func (h *operationsHandler) getInventory(c *gin.Context) {
	if !h.ready(c, h.inventory != nil) {
		return
	}
	id, ok := pathID(c, "goods_id")
	if !ok {
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
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "msg": "adjustment reason is required"})
		return
	}
	resp, err := h.inventory.SetStock(c, &req)
	if err == nil {
		err = h.audit(c, "inventory_adjusted", "goods", strconv.Itoa(int(id)))
	}
	writeRPC(c, resp, err)
}
func (h *operationsHandler) inventoryAdjustments(c *gin.Context) {
	if !h.ready(c, h.inventory != nil) {
		return
	}
	id, ok := pathID(c, "goods_id")
	if !ok {
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
	writeRPC(c, resp, err)
}
func (h *operationsHandler) getOrder(c *gin.Context) {
	if !h.ready(c, h.orders != nil) {
		return
	}
	resp, err := h.orders.GetOrderBySn(c, &orderpb.OrderLookupRequest{OrderSn: strings.TrimSpace(c.Param("order_sn"))})
	writeRPC(c, resp, err)
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
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "msg": "invalid refund request"})
		return
	}
	actor, err := userIDFromClaims(c)
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	correlation, _ := c.Get(operationCorrelationKey)
	orderSN := strings.TrimSpace(c.Param("order_sn"))
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
func (h *operationsHandler) reconcilePayments(c *gin.Context) {
	if !h.ready(c, h.orders != nil) {
		return
	}
	p, s := page(c)
	resp, err := h.orders.ListPaymentEvents(c, &orderpb.PaymentEventListRequest{OrderSn: strings.TrimSpace(c.Query("order_sn")), Page: p, PageSize: s, MismatchesOnly: true})
	if err != nil {
		writeRPC(c, nil, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"checked": resp.GetTotal(), "mismatch_count": resp.GetMismatchCount(), "items": resp.GetData()})
}

func (h *operationsHandler) ready(c *gin.Context, ready bool) bool {
	if ready {
		return true
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"code": http.StatusServiceUnavailable, "msg": "business rpc client is not initialized"})
	return false
}
func (h *operationsHandler) bind(c *gin.Context, ready bool, dst any) bool {
	if !h.ready(c, ready) {
		return false
	}
	if err := c.ShouldBindJSON(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "msg": "invalid request"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "msg": "invalid resource id"})
		return 0, false
	}
	return int32(value), true
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
		c.JSON(http.StatusBadGateway, gin.H{"code": http.StatusBadGateway, "msg": "business operation failed"})
		return
	}
	c.JSON(http.StatusOK, response)
}
