package order

import (
	"encoding/json"
	"goshop/app/goshop/api/internal/domain/request"
	"goshop/app/goshop/api/internal/service"
	orderv1 "goshop/app/goshop/api/internal/service/order/v1"
	"goshop/app/pkg/code"
	gin2 "goshop/app/pkg/translator/gin"
	gcode "goshop/gmicro/code"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"
	"goshop/pkg/money"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type orderController struct {
	trans ut.Translator
	sf    service.ServiceFactory
}

type CartItemForm struct {
	GoodsID int32 `form:"goods_id" json:"goods_id" binding:"required,gt=0"`
	Nums    int32 `form:"nums" json:"nums" binding:"required,gt=0"`
	Checked bool  `form:"checked" json:"checked"`
}

type SubmitOrderForm struct {
	OrderSn string `form:"order_sn" json:"order_sn"`
	Address string `form:"address" json:"address" binding:"required,min=1,max=100"`
	Name    string `form:"name" json:"name" binding:"required,min=1,max=20"`
	Mobile  string `form:"mobile" json:"mobile" binding:"required,len=11"`
	Post    string `form:"post" json:"post" binding:"omitempty,max=20"`
}

type SimulatePayCallbackForm struct {
	OrderSn string                        `form:"order_sn" json:"order_sn" binding:"required"`
	PayType string                        `form:"pay_type" json:"pay_type"`
	TradeNo string                        `form:"trade_no" json:"trade_no"`
	Items   []SimulatePayCallbackItemForm `form:"items" json:"items"`
	Success *bool                         `form:"success" json:"success" binding:"required"`
}

type SimulatePayCallbackItemForm struct {
	GoodsID int32 `form:"goods_id" json:"goods_id" binding:"required,gt=0"`
	Num     int32 `form:"num" json:"num" binding:"required,gt=0"`
}

func NewOrderController(sf service.ServiceFactory, trans ut.Translator) *orderController {
	return &orderController{sf: sf, trans: trans}
}

func (oc *orderController) ListCartItems(ctx *gin.Context) {
	userID, orderSrv, err := oc.authenticatedOrderService(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	resp, err := orderSrv.CartItemList(ctx, userID)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	data := make([]gin.H, 0, len(resp.GetData()))
	for _, item := range resp.GetData() {
		if item == nil {
			continue
		}
		data = append(data, cartItemResponse(item))
	}

	core.WriteResponse(ctx, nil, gin.H{
		"total": resp.GetTotal(),
		"data":  data,
	})
}

func (oc *orderController) CreateCartItem(ctx *gin.Context) {
	userID, orderSrv, err := oc.authenticatedOrderService(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	var form CartItemForm
	if err := ctx.ShouldBind(&form); err != nil {
		gin2.HandleValidatorError(ctx, err, oc.trans)
		return
	}

	item, err := orderSrv.CreateCartItem(ctx, userID, &orderv1.CartItemRequest{
		GoodsID: form.GoodsID,
		Nums:    form.Nums,
		Checked: form.Checked,
	})
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, cartItemResponse(item))
}

func (oc *orderController) UpdateCartItem(ctx *gin.Context) {
	userID, orderSrv, err := oc.authenticatedOrderService(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	var form CartItemForm
	if err := ctx.ShouldBind(&form); err != nil {
		gin2.HandleValidatorError(ctx, err, oc.trans)
		return
	}

	if err := orderSrv.UpdateCartItem(ctx, userID, &orderv1.CartItemRequest{
		GoodsID: form.GoodsID,
		Nums:    form.Nums,
		Checked: form.Checked,
	}); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, gin.H{"ok": true})
}

func (oc *orderController) DeleteCartItem(ctx *gin.Context) {
	userID, orderSrv, err := oc.authenticatedOrderService(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	var uri request.CartItemURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		gin2.HandleValidatorError(ctx, err, oc.trans)
		return
	}

	if err := orderSrv.DeleteCartItem(ctx, userID, uint64(uri.ID)); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, gin.H{"ok": true})
}

func (oc *orderController) SubmitOrder(ctx *gin.Context) {
	userID, orderSrv, err := oc.authenticatedOrderService(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	var form SubmitOrderForm
	if err := ctx.ShouldBind(&form); err != nil {
		gin2.HandleValidatorError(ctx, err, oc.trans)
		return
	}

	orderSn, err := orderSrv.SubmitOrder(ctx, userID, &orderv1.SubmitOrderRequest{
		OrderSn: form.OrderSn,
		Address: form.Address,
		Name:    form.Name,
		Mobile:  form.Mobile,
		Post:    form.Post,
	})
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, gin.H{"order_sn": orderSn})
}

func (oc *orderController) OrderList(ctx *gin.Context) {
	userID, orderSrv, err := oc.authenticatedOrderService(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	var query request.OrderListQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		gin2.HandleValidatorError(ctx, err, oc.trans)
		return
	}

	resp, err := orderSrv.OrderList(ctx, userID, &orderv1.OrderListFilter{
		Pages:       query.Pages,
		PagePerNums: query.PagePerNums,
	})
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	data := make([]gin.H, 0, len(resp.GetData()))
	for _, item := range resp.GetData() {
		if item == nil {
			continue
		}
		data = append(data, orderInfoResponse(item))
	}

	core.WriteResponse(ctx, nil, gin.H{
		"total": resp.GetTotal(),
		"data":  data,
	})
}

func (oc *orderController) OrderDetail(ctx *gin.Context) {
	userID, orderSrv, err := oc.authenticatedOrderService(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	var uri request.OrderDetailURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		gin2.HandleValidatorError(ctx, err, oc.trans)
		return
	}

	resp, err := orderSrv.OrderDetail(ctx, userID, uri.OrderSn)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	goods := make([]gin.H, 0, len(resp.GetGoods()))
	for _, item := range resp.GetGoods() {
		if item == nil {
			continue
		}
		goods = append(goods, orderItemResponse(item))
	}

	core.WriteResponse(ctx, nil, gin.H{
		"order_info": orderInfoResponse(resp.GetOrderInfo()),
		"goods":      goods,
	})
}

func (oc *orderController) OrderStatusLogs(ctx *gin.Context) {
	userID, orderSrv, err := oc.authenticatedOrderService(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	var uri request.OrderStatusLogsURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		gin2.HandleValidatorError(ctx, err, oc.trans)
		return
	}

	resp, err := orderSrv.OrderStatusLogs(ctx, userID, uri.OrderSn)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	data := make([]gin.H, 0, len(resp.GetData()))
	for _, item := range resp.GetData() {
		if item == nil {
			continue
		}
		data = append(data, orderStatusLogResponse(item))
	}

	core.WriteResponse(ctx, nil, gin.H{
		"total": resp.GetTotal(),
		"data":  data,
	})
}

func (oc *orderController) SimulatePayCallback(ctx *gin.Context) {
	var form SimulatePayCallbackForm
	if err := ctx.ShouldBind(&form); err != nil {
		gin2.HandleValidatorError(ctx, err, oc.trans)
		return
	}

	userID, orderSrv, err := oc.authenticatedOrderService(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	items := make([]orderv1.OrderItem, 0, len(form.Items))
	for _, item := range form.Items {
		items = append(items, orderv1.OrderItem{
			GoodsID: item.GoodsID,
			Num:     item.Num,
		})
	}

	if err := orderSrv.SimulatePayCallback(ctx, &orderv1.PayCallbackRequest{
		UserID:  userID,
		OrderSn: form.OrderSn,
		PayType: form.PayType,
		TradeNo: form.TradeNo,
		Items:   items,
		Success: form.Success != nil && *form.Success,
	}); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, gin.H{"ok": true})
}

func (oc *orderController) authenticatedOrderService(ctx *gin.Context) (uint64, orderv1.OrderSrv, error) {
	if oc == nil || oc.sf == nil {
		return 0, nil, errors.WithCode(code.ErrConnectGRPC, "order service is not initialized")
	}

	userID, err := userIDFromContext(ctx)
	if err != nil {
		return 0, nil, err
	}

	orderSrv := oc.sf.Orders()
	if orderSrv == nil {
		return 0, nil, errors.WithCode(code.ErrConnectGRPC, "order service is not initialized")
	}
	return userID, orderSrv, nil
}

func userIDFromContext(ctx *gin.Context) (uint64, error) {
	userID, ok := ctx.Get(middlewares.KeyUserID)
	if !ok {
		return 0, errors.WithCode(gcode.ErrInvalidAuthHeader, "user id is missing")
	}
	userIDFloat, ok := userID.(float64)
	if !ok {
		return 0, errors.WithCode(gcode.ErrInvalidAuthHeader, "user id has invalid type")
	}
	return uint64(userIDFloat), nil
}

func cartItemResponse(item interface {
	GetId() int32
	GetUserId() int32
	GetGoodsId() int32
	GetNums() int32
	GetChecked() bool
}) gin.H {
	return gin.H{
		"id":       item.GetId(),
		"user_id":  item.GetUserId(),
		"goods_id": item.GetGoodsId(),
		"nums":     item.GetNums(),
		"checked":  item.GetChecked(),
	}
}

func orderInfoResponse(item interface {
	GetId() int32
	GetUserId() int32
	GetOrderSn() string
	GetPayType() string
	GetStatus() string
	GetPost() string
	GetTotalFen() int64
	GetAddress() string
	GetName() string
	GetMobile() string
	GetAddTime() string
	GetTradeNo() string
	GetPayTime() int64
}) gin.H {
	return gin.H{
		"id":         item.GetId(),
		"user_id":    item.GetUserId(),
		"order_sn":   item.GetOrderSn(),
		"pay_type":   item.GetPayType(),
		"status":     item.GetStatus(),
		"post":       item.GetPost(),
		"total_fen":  item.GetTotalFen(),
		"total_yuan": money.NewFen(item.GetTotalFen()).YuanString(),
		"address":    item.GetAddress(),
		"name":       item.GetName(),
		"mobile":     item.GetMobile(),
		"add_time":   item.GetAddTime(),
		"trade_no":   item.GetTradeNo(),
		"pay_time":   item.GetPayTime(),
	}
}

func orderItemResponse(item interface {
	GetId() int32
	GetOrderId() int32
	GetGoodsId() int32
	GetGoodsName() string
	GetGoodsImage() string
	GetGoodsPriceFen() int64
	GetNums() int32
}) gin.H {
	return gin.H{
		"id":               item.GetId(),
		"order_id":         item.GetOrderId(),
		"goods_id":         item.GetGoodsId(),
		"goods_name":       item.GetGoodsName(),
		"goods_image":      item.GetGoodsImage(),
		"goods_price_fen":  item.GetGoodsPriceFen(),
		"goods_price_yuan": money.NewFen(item.GetGoodsPriceFen()).YuanString(),
		"nums":             item.GetNums(),
	}
}

func orderStatusLogResponse(item interface {
	GetId() int32
	GetOrderId() int32
	GetOrderSn() string
	GetFromStatus() string
	GetToStatus() string
	GetReason() string
	GetSource() string
	GetOperator() string
	GetAddTime() string
}) gin.H {
	return gin.H{
		"id":          item.GetId(),
		"order_id":    item.GetOrderId(),
		"order_sn":    item.GetOrderSn(),
		"from_status": item.GetFromStatus(),
		"to_status":   item.GetToStatus(),
		"reason":      item.GetReason(),
		"source":      item.GetSource(),
		"operator":    item.GetOperator(),
		"add_time":    item.GetAddTime(),
	}
}

func decodeJSONBody(data []byte) (map[string]any, error) {
	var body map[string]any
	err := json.Unmarshal(data, &body)
	return body, err
}
