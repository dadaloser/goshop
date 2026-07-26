package admin

import (
	"net/http"
	"strconv"
	"strings"

	goodspb "goshop/api/goods/v1"
	orderpb "goshop/api/order/v1"
	"goshop/app/pkg/authz"
	gauth "goshop/gmicro/server/restserver/middlewares/auth"

	"github.com/gin-gonic/gin"
)

func requestedOrderDomain(c *gin.Context) authz.BusinessDomain {
	roles := stringSet(gauth.ExtractClaims(c)["roles"])
	if roles[string(authz.StaffRoleFinance)] {
		return authz.BusinessDomainFinance
	}
	if roles[string(authz.StaffRoleOps)] {
		return authz.BusinessDomainOps
	}
	return authz.BusinessDomainSupport
}

func goodsScope(item *goodspb.GoodsInfoResponse, domain authz.BusinessDomain) authz.ResourceScope {
	if item == nil {
		return authz.ResourceScope{Domain: string(domain)}
	}
	return authz.ResourceScope{
		Domain:       string(domain),
		StoreID:      strings.TrimSpace(item.GetStoreId()),
		ResourceType: "goods",
		ResourceID:   strconv.Itoa(int(item.GetId())),
	}
}

func orderScope(item *orderpb.OrderInfoDetailResponse, domain authz.BusinessDomain) authz.ResourceScope {
	if item == nil || item.GetOrderInfo() == nil {
		return authz.ResourceScope{Domain: string(domain)}
	}
	return authz.ResourceScope{
		Domain:       string(domain),
		StoreID:      strings.TrimSpace(item.GetOrderInfo().GetStoreId()),
		ResourceType: "order",
		ResourceID:   strings.TrimSpace(item.GetOrderInfo().GetOrderSn()),
	}
}

func denyScope(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "msg": "resource scope denied"})
}

func (h *operationsHandler) authorizeGoods(c *gin.Context, goodsID int32, domain authz.BusinessDomain) (*goodspb.GoodsInfoResponse, bool) {
	if !h.ready(c, h.goods != nil) {
		return nil, false
	}
	resp, err := h.goods.GetGoodsDetail(c, &goodspb.GoodInfoRequest{Id: goodsID})
	if err != nil {
		writeRPC(c, nil, err)
		return nil, false
	}
	scope := goodsScope(resp, domain)
	if !claimsAllowScope(gauth.ExtractClaims(c), scope) {
		denyScope(c)
		return nil, false
	}
	return resp, true
}

func (h *operationsHandler) authorizeOrder(c *gin.Context, orderSN string, domain authz.BusinessDomain) (*orderpb.OrderInfoDetailResponse, bool) {
	if !h.ready(c, h.orders != nil) {
		return nil, false
	}
	resp, err := h.orders.GetOrderBySn(c, &orderpb.OrderLookupRequest{OrderSn: strings.TrimSpace(orderSN)})
	if err != nil {
		writeRPC(c, nil, err)
		return nil, false
	}
	scope := orderScope(resp, domain)
	if !claimsAllowScope(gauth.ExtractClaims(c), scope) {
		denyScope(c)
		return nil, false
	}
	return resp, true
}
