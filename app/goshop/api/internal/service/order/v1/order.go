package v1

import (
	"context"
	"strings"
	"time"

	opb "goshop/api/order/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"
)

type PayCallbackRequest struct {
	UserID  uint64
	OrderSn string
	PayType string
	TradeNo string
	Success bool
}

type OrderSrv interface {
	SimulatePayCallback(ctx context.Context, req *PayCallbackRequest) error
}

type orderService struct {
	data data.DataFactory
}

func NewOrderService(data data.DataFactory) OrderSrv {
	return &orderService{data: data}
}

func (os *orderService) SimulatePayCallback(ctx context.Context, req *PayCallbackRequest) error {
	if os == nil || os.data == nil {
		return errors.WithCode(code.ErrConnectGRPC, "order data client is not initialized")
	}
	if req == nil {
		return errors.WithCode(code.ErrOrderStatusInvalid, "pay callback request is required")
	}
	req.OrderSn = strings.TrimSpace(req.OrderSn)
	req.PayType = strings.TrimSpace(req.PayType)
	req.TradeNo = strings.TrimSpace(req.TradeNo)
	if req.UserID == 0 || req.OrderSn == "" {
		return errors.WithCode(code.ErrOrderStatusInvalid, "user_id and order_sn are required")
	}

	client := os.data.Orders()
	if client == nil {
		return errors.WithCode(code.ErrConnectGRPC, "order grpc client is not initialized")
	}

	status := &opb.OrderStatus{
		OrderSn: req.OrderSn,
	}
	if req.Success {
		if req.TradeNo == "" {
			return errors.WithCode(code.ErrOrderStatusInvalid, "trade_no is required for paid callback")
		}
		status.Status = "TRADE_SUCCESS"
		status.PayType = req.PayType
		status.TradeNo = req.TradeNo
		status.PayTime = time.Now().Unix()
	} else {
		status.Status = "TRADE_CLOSED"
	}

	_, err := client.UpdateOrderStatus(ctx, status)
	return err
}

var _ OrderSrv = &orderService{}
