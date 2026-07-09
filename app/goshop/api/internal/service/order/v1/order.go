package v1

import (
	"context"
	"strings"
	"time"

	ipb "goshop/api/inventory/v1"
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
	Items   []OrderItem
	Success bool
}

type OrderItem struct {
	GoodsID int32
	Num     int32
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
	inventoryClient := os.data.Inventory()
	if inventoryClient == nil {
		return errors.WithCode(code.ErrConnectGRPC, "inventory grpc client is not initialized")
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

	if _, err := client.UpdateOrderStatus(ctx, status); err != nil {
		return err
	}

	sellInfo := &ipb.SellInfo{OrderSn: req.OrderSn}
	for _, item := range req.Items {
		if item.GoodsID <= 0 || item.Num <= 0 {
			return errors.WithCode(code.ErrOrderStatusInvalid, "pay callback items are invalid")
		}
		sellInfo.GoodsInfo = append(sellInfo.GoodsInfo, &ipb.GoodsInvInfo{
			GoodsId: item.GoodsID,
			Num:     item.Num,
		})
	}
	if len(sellInfo.GoodsInfo) == 0 {
		return errors.WithCode(code.ErrOrderStatusInvalid, "pay callback items are required")
	}

	if req.Success {
		_, err := inventoryClient.Confirm(ctx, sellInfo)
		return err
	}
	_, err := inventoryClient.Release(ctx, sellInfo)
	return err
}

var _ OrderSrv = &orderService{}
