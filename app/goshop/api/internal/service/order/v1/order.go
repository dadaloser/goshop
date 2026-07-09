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

const (
	orderStatusPaying       = "PAYING"
	orderStatusWaitBuyerPay = "WAIT_BUYER_PAY"
	orderStatusTradeSuccess = "TRADE_SUCCESS"
	orderStatusTradeClosed  = "TRADE_CLOSED"
	orderStatusTradeFinish  = "TRADE_FINISHED"
)

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

	orderDetail, err := client.OrderDetail(ctx, &opb.OrderRequest{
		UserId:  int32(req.UserID),
		OrderSn: req.OrderSn,
	})
	if err != nil {
		return err
	}
	if orderDetail == nil || orderDetail.OrderInfo == nil {
		return errors.WithCode(code.ErrOrderNotFound, "order not found")
	}

	targetStatus := orderStatusTradeClosed
	if req.Success {
		if req.TradeNo == "" {
			return errors.WithCode(code.ErrOrderStatusInvalid, "trade_no is required for paid callback")
		}
		targetStatus = orderStatusTradeSuccess
	}

	currentStatus := strings.TrimSpace(orderDetail.OrderInfo.Status)
	if !canApplyPayCallback(currentStatus, targetStatus) {
		return errors.WithCode(code.ErrOrderStatusInvalid, "invalid order status transition")
	}

	sellInfo, err := sellInfoFromOrder(req.OrderSn, orderDetail.Goods)
	if err != nil {
		return err
	}

	if req.Success {
		if _, err := inventoryClient.Confirm(ctx, sellInfo); err != nil {
			return err
		}
	} else {
		if _, err := inventoryClient.Release(ctx, sellInfo); err != nil {
			return err
		}
	}

	status := &opb.OrderStatus{
		OrderSn: req.OrderSn,
		Status:  targetStatus,
	}
	if req.Success {
		status.PayType = req.PayType
		status.TradeNo = req.TradeNo
		status.PayTime = time.Now().Unix()
	}

	_, err = client.UpdateOrderStatus(ctx, status)
	return err
}

var _ OrderSrv = &orderService{}

func sellInfoFromOrder(orderSn string, goods []*opb.OrderItemResponse) (*ipb.SellInfo, error) {
	sellInfo := &ipb.SellInfo{OrderSn: orderSn}
	for _, item := range goods {
		if item == nil || item.GoodsId <= 0 || item.Nums <= 0 {
			return nil, errors.WithCode(code.ErrOrderStatusInvalid, "order goods are invalid")
		}
		sellInfo.GoodsInfo = append(sellInfo.GoodsInfo, &ipb.GoodsInvInfo{
			GoodsId: item.GoodsId,
			Num:     item.Nums,
		})
	}
	if len(sellInfo.GoodsInfo) == 0 {
		return nil, errors.WithCode(code.ErrOrderStatusInvalid, "order goods are required")
	}
	return sellInfo, nil
}

func canApplyPayCallback(currentStatus, targetStatus string) bool {
	currentStatus = strings.TrimSpace(currentStatus)
	targetStatus = strings.TrimSpace(targetStatus)
	if currentStatus == "" || currentStatus == targetStatus {
		return true
	}

	switch currentStatus {
	case orderStatusWaitBuyerPay:
		return targetStatus == orderStatusPaying || targetStatus == orderStatusTradeSuccess || targetStatus == orderStatusTradeClosed
	case orderStatusPaying:
		return targetStatus == orderStatusTradeSuccess || targetStatus == orderStatusTradeClosed
	case orderStatusTradeSuccess:
		return targetStatus == orderStatusTradeFinish
	case orderStatusTradeClosed, orderStatusTradeFinish:
		return false
	default:
		return false
	}
}
