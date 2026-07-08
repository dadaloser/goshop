package order

import (
	"goshop/app/goshop/api/internal/service"
	orderv1 "goshop/app/goshop/api/internal/service/order/v1"
	"goshop/app/pkg/code"
	gin2 "goshop/app/pkg/translator/gin"
	gcode "goshop/gmicro/code"
	"goshop/gmicro/server/restserver/middlewares"
	"goshop/pkg/common/core"
	"goshop/pkg/errors"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type orderController struct {
	trans ut.Translator
	sf    service.ServiceFactory
}

type SimulatePayCallbackForm struct {
	OrderSn string `form:"order_sn" json:"order_sn" binding:"required"`
	PayType string `form:"pay_type" json:"pay_type"`
	TradeNo string `form:"trade_no" json:"trade_no"`
	Success *bool  `form:"success" json:"success" binding:"required"`
}

func NewOrderController(sf service.ServiceFactory, trans ut.Translator) *orderController {
	return &orderController{sf: sf, trans: trans}
}

func (oc *orderController) SimulatePayCallback(ctx *gin.Context) {
	if oc == nil || oc.sf == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "order service is not initialized"), nil)
		return
	}

	var form SimulatePayCallbackForm
	if err := ctx.ShouldBind(&form); err != nil {
		gin2.HandleValidatorError(ctx, err, oc.trans)
		return
	}

	userID, err := userIDFromContext(ctx)
	if err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	orderSrv := oc.sf.Orders()
	if orderSrv == nil {
		core.WriteResponse(ctx, errors.WithCode(code.ErrConnectGRPC, "order service is not initialized"), nil)
		return
	}

	if err := orderSrv.SimulatePayCallback(ctx, &orderv1.PayCallbackRequest{
		UserID:  userID,
		OrderSn: form.OrderSn,
		PayType: form.PayType,
		TradeNo: form.TradeNo,
		Success: form.Success != nil && *form.Success,
	}); err != nil {
		core.WriteResponse(ctx, err, nil)
		return
	}

	core.WriteResponse(ctx, nil, gin.H{"ok": true})
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
