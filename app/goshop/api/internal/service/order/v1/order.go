package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	ipb "goshop/api/inventory/v1"
	opb "goshop/api/order/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/goshop/api/internal/payment"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	"goshop/pkg/errors"
	"goshop/pkg/log"
)

type PayCallbackRequest struct {
	Provider  string
	EventID   string
	EventType string
	AmountFen int64
	UserID    uint64
	OrderSn   string
	PayType   string
	TradeNo   string
	Items     []OrderItem
	Success   bool
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
	OrderStatusLogs(ctx context.Context, userID uint64, orderSn string) (*opb.OrderStatusLogListResponse, error)
	SimulatePayCallback(ctx context.Context, req *PayCallbackRequest) error
}

type orderService struct {
	data        data.DataFactory
	provider    payment.Provider
	paymentOpts *options.PaymentOptions
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

func NewOrderServiceWithPayment(data data.DataFactory, opts *options.PaymentOptions) OrderSrv {
	return &orderService{data: data, provider: payment.NewProvider(opts), paymentOpts: opts}
}

type PaymentInitiation struct {
	PaymentID, Provider, CheckoutURL string
	ExpiresAt, AmountFen             int64
}

func (os *orderService) InitiatePayment(ctx context.Context, userID uint64, orderSN string) (*PaymentInitiation, error) {
	if os == nil || os.paymentOpts == nil || !os.paymentOpts.Enabled {
		return nil, errors.WithCode(code.ErrConnectGRPC, "payment provider is disabled")
	}
	detail, err := os.OrderDetail(ctx, userID, orderSN)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(detail.GetOrderInfo().GetStatus())
	if status != orderStatusWaitBuyerPay && status != orderStatusPaying {
		return nil, errors.WithCode(code.ErrOrderStatusInvalid, "order cannot be paid in current status")
	}
	amount := detail.GetOrderInfo().GetTotalFen()
	result, err := os.provider.Initiate(ctx, payment.InitiateRequest{OrderSN: orderSN, AmountFen: amount, Subject: "goshop order"})
	if err != nil {
		return nil, err
	}
	if _, err = os.data.Orders().UpdateOrderStatus(ctx, &opb.OrderStatus{OrderSn: orderSN, Status: orderStatusPaying}); err != nil {
		return nil, err
	}
	return &PaymentInitiation{PaymentID: result.PaymentID, Provider: result.Provider, CheckoutURL: result.CheckoutURL, ExpiresAt: result.ExpiresAt.Unix(), AmountFen: amount}, nil
}

func (os *orderService) CancelOrder(ctx context.Context, userID uint64, orderSN string) error {
	detail, err := os.OrderDetail(ctx, userID, orderSN)
	if err != nil {
		return err
	}
	current := strings.TrimSpace(detail.GetOrderInfo().GetStatus())
	if current != orderStatusWaitBuyerPay && current != orderStatusPaying {
		return errors.WithCode(code.ErrOrderStatusInvalid, "order cannot be cancelled in current status")
	}
	sellInfo, err := sellInfoFromOrder(orderSN, detail.GetGoods())
	if err != nil {
		return err
	}
	if _, err = os.data.Inventory().Release(ctx, sellInfo); err != nil {
		return err
	}
	_, err = os.data.Orders().UpdateOrderStatus(ctx, &opb.OrderStatus{OrderSn: orderSN, Status: orderStatusTradeClosed, Reason: "cancelled by customer"})
	return err
}

func (os *orderService) ProcessPayCallback(ctx context.Context, req *payment.CallbackRequest) (bool, error) {
	if req == nil || req.Provider == "" || req.EventID == "" || req.EventType == "" || req.OrderSN == "" || req.AmountFen < 0 {
		return false, errors.WithCode(code.ErrOrderStatusInvalid, "payment callback is invalid")
	}
	orders := os.data.Orders()
	refundAmount := int64(0)
	if req.EventType == "refund_succeeded" {
		refundAmount = req.AmountFen
	}
	begin, err := orders.BeginPaymentEvent(ctx, &opb.PaymentEventRequest{Provider: req.Provider, EventId: req.EventID, OrderSn: req.OrderSN, TradeNo: req.TradeNo, EventType: req.EventType, ProviderAmountFen: req.AmountFen, RefundAmountFen: refundAmount})
	if err != nil {
		return false, err
	}
	if !begin.GetAccepted() {
		if begin.GetCompleted() {
			return true, nil
		}
		return false, errors.WithCode(code.ErrConnectGRPC, "payment callback is already processing")
	}
	success := false
	detail := "callback processing failed"
	defer func() {
		if _, completeErr := orders.CompletePaymentEvent(context.WithoutCancel(ctx), &opb.CompletePaymentEventRequest{Id: begin.GetId(), Success: success, ErrorDetail: detail}); completeErr != nil {
			log.Errorf("complete payment event failed: event_id=%s error=%v", req.EventID, completeErr)
		}
	}()
	amountMismatch := req.EventType != "refund_succeeded" && req.AmountFen != begin.GetOrderAmountFen()
	invalidRefund := req.EventType == "refund_succeeded" && (req.AmountFen <= 0 || req.AmountFen > begin.GetOrderAmountFen())
	if amountMismatch || invalidRefund {
		detail = "payment amount mismatch"
		return false, errors.WithCode(code.ErrOrderStatusInvalid, detail)
	}
	orderDetail, err := orders.GetOrderBySn(ctx, &opb.OrderLookupRequest{OrderSn: req.OrderSN})
	if err != nil {
		detail = "load order failed"
		return false, err
	}
	currentStatus := strings.TrimSpace(orderDetail.GetOrderInfo().GetStatus())
	if !paymentEventAllowed(req.EventType, currentStatus) {
		detail = "payment event is out of order"
		return false, errors.WithCode(code.ErrOrderStatusInvalid, detail)
	}
	sellInfo, err := sellInfoFromOrder(req.OrderSN, orderDetail.GetGoods())
	if err != nil {
		detail = "invalid order items"
		return false, err
	}
	target := ""
	switch req.EventType {
	case "payment_succeeded":
		target = orderStatusTradeSuccess
		if _, err = os.data.Inventory().Confirm(ctx, sellInfo); err != nil {
			detail = "confirm inventory failed"
			return false, err
		}
	case "payment_failed":
		target = orderStatusTradeClosed
		if _, err = os.data.Inventory().Release(ctx, sellInfo); err != nil {
			detail = "release inventory failed"
			return false, err
		}
	case "refund_succeeded":
		target = "REFUNDED"
	default:
		detail = "unknown payment event"
		return false, errors.WithCode(code.ErrOrderStatusInvalid, detail)
	}
	status := &opb.OrderStatus{OrderSn: req.OrderSN, Status: target, PayType: req.Provider, TradeNo: req.TradeNo}
	if target == orderStatusTradeSuccess {
		status.PayTime = time.Now().Unix()
	}
	if _, err = orders.UpdateOrderStatus(ctx, status); err != nil {
		detail = "update order status failed"
		return false, err
	}
	success = true
	detail = ""
	return false, nil
}

func paymentEventAllowed(eventType, status string) bool {
	switch eventType {
	case "payment_succeeded", "payment_failed":
		return status == orderStatusWaitBuyerPay || status == orderStatusPaying
	case "refund_succeeded":
		return status == "REFUND_PENDING"
	default:
		return false
	}
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

func (os *orderService) OrderStatusLogs(ctx context.Context, userID uint64, orderSn string) (*opb.OrderStatusLogListResponse, error) {
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

	resp, err := client.OrderStatusLogs(ctx, &opb.OrderRequest{
		UserId:  int32(userID),
		OrderSn: orderSn,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "order grpc response is empty")
	}
	return resp, nil
}

func (os *orderService) SimulatePayCallback(ctx context.Context, req *PayCallbackRequest) error {
	startedAt := time.Now()
	targetStatus := orderStatusTradeClosed
	var callbackErr error
	defer func() {
		observePayCallback(targetStatus, payCallbackMetricResult(callbackErr), startedAt)
	}()

	if os == nil || os.data == nil {
		callbackErr = errors.WithCode(code.ErrConnectGRPC, "order data client is not initialized")
		return callbackErr
	}
	if req == nil {
		callbackErr = errors.WithCode(code.ErrOrderStatusInvalid, "pay callback request is required")
		return callbackErr
	}
	req.OrderSn = strings.TrimSpace(req.OrderSn)
	req.PayType = strings.TrimSpace(req.PayType)
	req.TradeNo = strings.TrimSpace(req.TradeNo)
	if req.UserID == 0 || req.OrderSn == "" {
		callbackErr = errors.WithCode(code.ErrOrderStatusInvalid, "user_id and order_sn are required")
		return callbackErr
	}

	client := os.data.Orders()
	if client == nil {
		callbackErr = errors.WithCode(code.ErrConnectGRPC, "order grpc client is not initialized")
		return callbackErr
	}
	inventoryClient := os.data.Inventory()
	if inventoryClient == nil {
		callbackErr = errors.WithCode(code.ErrConnectGRPC, "inventory grpc client is not initialized")
		return callbackErr
	}

	orderDetail, err := client.OrderDetail(ctx, &opb.OrderRequest{
		UserId:  int32(req.UserID),
		OrderSn: req.OrderSn,
	})
	if err != nil {
		callbackErr = err
		return callbackErr
	}
	if orderDetail == nil || orderDetail.OrderInfo == nil {
		callbackErr = errors.WithCode(code.ErrOrderNotFound, "order not found")
		return callbackErr
	}

	if req.Success {
		if req.TradeNo == "" {
			callbackErr = errors.WithCode(code.ErrOrderStatusInvalid, "trade_no is required for paid callback")
			return callbackErr
		}
		targetStatus = orderStatusTradeSuccess
	}

	currentStatus := strings.TrimSpace(orderDetail.OrderInfo.Status)
	if !canApplyPayCallback(currentStatus, targetStatus) {
		callbackErr = errors.WithCode(code.ErrOrderStatusInvalid, "invalid order status transition")
		return callbackErr
	}

	sellInfo, err := sellInfoFromOrder(req.OrderSn, orderDetail.Goods)
	if err != nil {
		callbackErr = err
		return callbackErr
	}

	if req.Success {
		if _, err := inventoryClient.Confirm(ctx, sellInfo); err != nil {
			callbackErr = err
			return callbackErr
		}
	} else {
		if _, err := inventoryClient.Release(ctx, sellInfo); err != nil {
			callbackErr = err
			return callbackErr
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
	if err != nil {
		callbackErr = err
		return callbackErr
	}

	log.InfoC(ctx, "order pay callback processed",
		log.Uint64(log.KeyUserID, req.UserID),
		log.String(log.KeyOrderSN, req.OrderSn),
		log.String(log.KeyCurrentStatus, currentStatus),
		log.String(log.KeyTargetStatus, targetStatus),
		log.Bool(log.KeySuccess, req.Success),
		log.String(log.KeyPayType, req.PayType),
	)
	return nil
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
