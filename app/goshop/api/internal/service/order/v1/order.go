package v1

import (
	"context"
	"fmt"
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

type CartItemRequest struct {
	GoodsID int32
	Nums    int32
	Checked bool
}

type SubmitOrderRequest struct {
	OrderSn string
	Address string
	Name    string
	Mobile  string
	Post    string
}

type OrderListFilter struct {
	Pages       int32
	PagePerNums int32
}

type OrderSrv interface {
	CartItemList(ctx context.Context, userID uint64) (*opb.CartItemListResponse, error)
	CreateCartItem(ctx context.Context, userID uint64, req *CartItemRequest) (*opb.ShopCartInfoResponse, error)
	UpdateCartItem(ctx context.Context, userID uint64, req *CartItemRequest) error
	DeleteCartItem(ctx context.Context, userID, id uint64) error
	SubmitOrder(ctx context.Context, userID uint64, req *SubmitOrderRequest) (string, error)
	OrderList(ctx context.Context, userID uint64, filter *OrderListFilter) (*opb.OrderListResponse, error)
	OrderDetail(ctx context.Context, userID uint64, orderSn string) (*opb.OrderInfoDetailResponse, error)
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

	defaultOrderPage     = int32(1)
	defaultOrderPageSize = int32(10)
)

func NewOrderService(data data.DataFactory) OrderSrv {
	return &orderService{data: data}
}

func (os *orderService) CartItemList(ctx context.Context, userID uint64) (*opb.CartItemListResponse, error) {
	if os == nil || os.data == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order data client is not initialized")
	}
	if userID == 0 {
		return &opb.CartItemListResponse{}, nil
	}

	client := os.data.Orders()
	if client == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order grpc client is not initialized")
	}

	resp, err := client.CartItemList(ctx, &opb.UserInfo{Id: int32(userID)})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order grpc response is empty")
	}
	return resp, nil
}

func (os *orderService) CreateCartItem(ctx context.Context, userID uint64, req *CartItemRequest) (*opb.ShopCartInfoResponse, error) {
	if os == nil || os.data == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order data client is not initialized")
	}
	if userID == 0 || req == nil || req.GoodsID <= 0 || req.Nums <= 0 {
		return nil, errors.WithCode(code.ErrShopCartItemNotFound, "shop cart item not found")
	}

	client := os.data.Orders()
	if client == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order grpc client is not initialized")
	}

	resp, err := client.CreateCartItem(ctx, &opb.CartItemRequest{
		UserId:  int32(userID),
		GoodsId: req.GoodsID,
		Nums:    req.Nums,
		Checked: req.Checked,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order grpc response is empty")
	}
	return resp, nil
}

func (os *orderService) UpdateCartItem(ctx context.Context, userID uint64, req *CartItemRequest) error {
	if os == nil || os.data == nil {
		return errors.WithCode(code.ErrConnectGRPC, "order data client is not initialized")
	}
	if userID == 0 || req == nil || req.GoodsID <= 0 || req.Nums <= 0 {
		return errors.WithCode(code.ErrShopCartItemNotFound, "shop cart item not found")
	}

	client := os.data.Orders()
	if client == nil {
		return errors.WithCode(code.ErrConnectGRPC, "order grpc client is not initialized")
	}

	_, err := client.UpdateCartItem(ctx, &opb.CartItemRequest{
		UserId:  int32(userID),
		GoodsId: req.GoodsID,
		Nums:    req.Nums,
		Checked: req.Checked,
	})
	return err
}

func (os *orderService) DeleteCartItem(ctx context.Context, userID, id uint64) error {
	if os == nil || os.data == nil {
		return errors.WithCode(code.ErrConnectGRPC, "order data client is not initialized")
	}
	if userID == 0 || id == 0 {
		return errors.WithCode(code.ErrShopCartItemNotFound, "shop cart item not found")
	}

	client := os.data.Orders()
	if client == nil {
		return errors.WithCode(code.ErrConnectGRPC, "order grpc client is not initialized")
	}

	_, err := client.DeleteCartItem(ctx, &opb.CartItemRequest{
		Id:     int32(id),
		UserId: int32(userID),
	})
	return err
}

func (os *orderService) SubmitOrder(ctx context.Context, userID uint64, req *SubmitOrderRequest) (string, error) {
	if os == nil || os.data == nil {
		return "", errors.WithCode(code.ErrConnectGRPC, "order data client is not initialized")
	}
	if userID == 0 || req == nil {
		return "", errors.WithCode(code.ErrSubmitOrder, "order request is required")
	}

	req.Address = strings.TrimSpace(req.Address)
	req.Name = strings.TrimSpace(req.Name)
	req.Mobile = strings.TrimSpace(req.Mobile)
	req.Post = strings.TrimSpace(req.Post)
	req.OrderSn = strings.TrimSpace(req.OrderSn)
	if req.Address == "" || req.Name == "" || req.Mobile == "" {
		return "", errors.WithCode(code.ErrSubmitOrder, "address, name and mobile are required")
	}
	if req.OrderSn == "" {
		req.OrderSn = generateOrderSn(userID)
	}

	client := os.data.Orders()
	if client == nil {
		return "", errors.WithCode(code.ErrConnectGRPC, "order grpc client is not initialized")
	}

	_, err := client.SubmitOrder(ctx, &opb.OrderRequest{
		UserId:  int32(userID),
		Address: req.Address,
		Name:    req.Name,
		Mobile:  req.Mobile,
		Post:    req.Post,
		OrderSn: req.OrderSn,
	})
	if err != nil {
		return "", err
	}
	return req.OrderSn, nil
}

func (os *orderService) OrderList(ctx context.Context, userID uint64, filter *OrderListFilter) (*opb.OrderListResponse, error) {
	if os == nil || os.data == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order data client is not initialized")
	}
	if userID == 0 {
		return &opb.OrderListResponse{}, nil
	}

	client := os.data.Orders()
	if client == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order grpc client is not initialized")
	}

	filter = normalizeOrderListFilter(filter)
	resp, err := client.OrderList(ctx, &opb.OrderFilterRequest{
		UserId:      int32(userID),
		Pages:       filter.Pages,
		PagePerNums: filter.PagePerNums,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order grpc response is empty")
	}
	return resp, nil
}

func (os *orderService) OrderDetail(ctx context.Context, userID uint64, orderSn string) (*opb.OrderInfoDetailResponse, error) {
	if os == nil || os.data == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order data client is not initialized")
	}
	orderSn = strings.TrimSpace(orderSn)
	if userID == 0 || orderSn == "" {
		return nil, errors.WithCode(code.ErrOrderNotFound, "order not found")
	}

	client := os.data.Orders()
	if client == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order grpc client is not initialized")
	}

	resp, err := client.OrderDetail(ctx, &opb.OrderRequest{
		UserId:  int32(userID),
		OrderSn: orderSn,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order grpc response is empty")
	}
	if resp.OrderInfo == nil {
		return nil, errors.WithCode(code.ErrOrderNotFound, "order not found")
	}
	return resp, nil
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

func normalizeOrderListFilter(filter *OrderListFilter) *OrderListFilter {
	if filter == nil {
		return &OrderListFilter{
			Pages:       defaultOrderPage,
			PagePerNums: defaultOrderPageSize,
		}
	}
	if filter.Pages <= 0 {
		filter.Pages = defaultOrderPage
	}
	if filter.PagePerNums <= 0 {
		filter.PagePerNums = defaultOrderPageSize
	}
	return filter
}

func generateOrderSn(userID uint64) string {
	return fmt.Sprintf("%d%08d", time.Now().UnixNano(), userID%100000000)
}
