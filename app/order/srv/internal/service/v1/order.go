package service

import (
	"context"
	"fmt"
	proto "goshop/api/inventory/v1"
	proto3 "goshop/api/order/v1"
	v12 "goshop/app/order/srv/internal/data/v1"
	"goshop/app/order/srv/internal/domain/do"
	"goshop/app/order/srv/internal/domain/dto"
	"goshop/app/pkg/client"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	v1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"
	"goshop/pkg/log"
	"goshop/pkg/money"
	"strings"
	"time"

	"github.com/dtm-labs/client/dtmgrpc"
	"gorm.io/gorm"
)

var (
	inventory_busi = client.ServiceEndpoint(client.ServiceInventory)
	order_busi     = client.ServiceEndpoint(client.ServiceOrder)
)

const (
	OrderStatusPaying        = "PAYING"
	OrderStatusWaitBuyerPay  = "WAIT_BUYER_PAY"
	OrderStatusTradeSuccess  = "TRADE_SUCCESS"
	OrderStatusTradeClosed   = "TRADE_CLOSED"
	OrderStatusTradeFinished = "TRADE_FINISHED"
	OrderStatusRefundPending = "REFUND_PENDING"
	OrderStatusRefundFailed  = "REFUND_FAILED"
	OrderStatusRefunded      = "REFUNDED"
)

var validOrderStatuses = map[string]struct{}{
	OrderStatusPaying:        {},
	OrderStatusWaitBuyerPay:  {},
	OrderStatusTradeSuccess:  {},
	OrderStatusTradeClosed:   {},
	OrderStatusTradeFinished: {},
	OrderStatusRefundPending: {},
	OrderStatusRefundFailed:  {},
	OrderStatusRefunded:      {},
}

type OrderSrv interface {
	CartItemList(ctx context.Context, userID uint64, meta v1.ListMeta, orderBy []string) (*dto.ShopCartDTOList, error)
	CreateCartItem(ctx context.Context, cartItem *dto.ShopCartDTO) (*dto.ShopCartDTO, error)
	UpdateCartItem(ctx context.Context, cartItem *dto.ShopCartDTO) error
	DeleteCartItem(ctx context.Context, userID, id uint64) error
	Get(ctx context.Context, userID uint64, orderSn string) (*dto.OrderDTO, error)
	StatusLogs(ctx context.Context, userID uint64, orderSn string) (*dto.OrderStatusLogDTOList, error)
	List(ctx context.Context, userID uint64, meta v1.ListMeta, orderBy []string) (*dto.OrderDTOList, error)
	Submit(ctx context.Context, order *dto.OrderDTO) error
	Create(ctx context.Context, order *dto.OrderDTO) error
	CreateCom(ctx context.Context, order *dto.OrderDTO) error //这是create的补偿
	Update(ctx context.Context, order *dto.OrderDTO) error
}

type orderService struct {
	data     v12.DataFactory
	dtmOpts  *options.DtmOptions
	upstream upstream
}

func (os *orderService) CartItemList(ctx context.Context, userID uint64, meta v1.ListMeta, orderBy []string) (*dto.ShopCartDTOList, error) {
	if userID == 0 {
		return &dto.ShopCartDTOList{}, nil
	}

	carts, err := os.data.ShopCarts().List(ctx, userID, false, meta, orderBy)
	if err != nil {
		return nil, err
	}

	ret := &dto.ShopCartDTOList{TotalCount: carts.TotalCount}
	for _, item := range carts.Items {
		ret.Items = append(ret.Items, &dto.ShopCartDTO{ShoppingCartDO: *item})
	}
	return ret, nil
}

func (os *orderService) CreateCartItem(ctx context.Context, cartItem *dto.ShopCartDTO) (*dto.ShopCartDTO, error) {
	if cartItem == nil || cartItem.User <= 0 || cartItem.Goods <= 0 || cartItem.Nums <= 0 {
		return nil, errors.WithCode(code.ErrShopCartItemNotFound, "shop cart item not found")
	}

	existing, err := os.data.ShopCarts().Get(ctx, uint64(cartItem.User), uint64(cartItem.Goods))
	if err == nil {
		existing.Nums += cartItem.Nums
		existing.Checked = cartItem.Checked
		if err := os.data.ShopCarts().UpdateNum(ctx, existing); err != nil {
			return nil, err
		}
		return &dto.ShopCartDTO{ShoppingCartDO: *existing}, nil
	}
	if !errors.IsCode(err, code.ErrShopCartItemNotFound) {
		return nil, err
	}

	if err := os.data.ShopCarts().Create(ctx, &cartItem.ShoppingCartDO); err != nil {
		return nil, err
	}
	return cartItem, nil
}

func (os *orderService) UpdateCartItem(ctx context.Context, cartItem *dto.ShopCartDTO) error {
	if cartItem == nil || cartItem.User <= 0 || cartItem.Goods <= 0 || cartItem.Nums <= 0 {
		return errors.WithCode(code.ErrShopCartItemNotFound, "shop cart item not found")
	}
	return os.data.ShopCarts().UpdateNum(ctx, &cartItem.ShoppingCartDO)
}

func (os *orderService) DeleteCartItem(ctx context.Context, userID, id uint64) error {
	if userID == 0 || id == 0 {
		return errors.WithCode(code.ErrShopCartItemNotFound, "shop cart item not found")
	}
	return os.data.ShopCarts().Delete(ctx, userID, id)
}

func (os *orderService) CreateCom(ctx context.Context, order *dto.OrderDTO) error {
	if order == nil || strings.TrimSpace(order.OrderSn) == "" {
		return nil
	}

	existing, err := os.data.Orders().Get(ctx, order.OrderSn)
	if err != nil {
		if errors.IsCode(err, code.ErrOrderNotFound) {
			return nil
		}
		return err
	}

	txn := os.data.Begin()
	if txn == nil {
		if err := os.data.ShopCarts().RestoreCheckedItems(ctx, nil, uint64(existing.User), existing.OrderGoods); err != nil {
			return fmt.Errorf("restore selected shop carts: %w", err)
		}
		if err := os.data.Orders().DeleteByOrderSn(ctx, nil, existing.OrderSn); err != nil {
			return fmt.Errorf("delete compensated order: %w", err)
		}
		return nil
	}
	if txn.Error != nil {
		return fmt.Errorf("begin create order compensation transaction: %w", txn.Error)
	}

	committed := false
	defer func() {
		if !committed {
			_ = txn.Rollback()
		}
	}()

	err = os.data.ShopCarts().RestoreCheckedItems(ctx, txn, uint64(existing.User), existing.OrderGoods)
	if err != nil {
		return fmt.Errorf("restore selected shop carts: %w", err)
	}

	err = os.data.Orders().DeleteByOrderSn(ctx, txn, existing.OrderSn)
	if err != nil {
		return fmt.Errorf("delete compensated order: %w", err)
	}

	if err := txn.Commit().Error; err != nil {
		return fmt.Errorf("commit create order compensation transaction: %w", err)
	}
	committed = true
	return nil
}

func (os *orderService) Create(ctx context.Context, order *dto.OrderDTO) (err error) {
	/*
		1. 生成order_info表
		2. 生成order_goods表
		3. 根据order找到对应的购物车条目，删除购物车条目
	*/
	if order == nil {
		return errors.WithCode(code.ErrSubmitOrder, "order is required")
	}
	order.OrderSn = strings.TrimSpace(order.OrderSn)
	if order.OrderSn == "" {
		return errors.WithCode(code.ErrSubmitOrder, "order_sn is required")
	}
	if !hasOrderGoods(order.OrderGoods) {
		return errors.WithCode(code.ErrNoGoodsSelect, "没有选择商品")
	}
	order.Status = normalizeInitialOrderStatus(order.Status)

	existing, err := os.data.Orders().Get(ctx, order.OrderSn)
	if err == nil {
		if sameCreateOrder(existing, order) {
			return nil
		}
		return errors.WithCode(code.ErrOrderConflict, "order_sn already exists with different order data")
	}
	if !errors.IsCode(err, code.ErrOrderNotFound) {
		return err
	}

	var goodsIds []int32
	for _, value := range order.OrderGoods {
		goodsIds = append(goodsIds, value.Goods)
	}

	//获取goods信息
	goodsMap, err := os.upstream.goods.BatchGetGoods(ctx, goodsIds)
	if err != nil {
		log.Errorf("批量获取商品信息失败，goodids: %v, err:%v", goodsIds, err)
		return err
	}
	if len(goodsMap) != len(goodsIds) {
		log.Errorf("批量获取商品信息失败，goodids: %v, 返回值：%v, err:%v", goodsIds, goodsMap, err)
		return errors.WithCode(code.ErrGoodsNotFound, "商品不存在或者部分不存在")
	}

	//生成订单总金额
	var orderAmountFen money.Fen
	for _, value := range order.OrderGoods {
		goodsInfo := goodsMap[value.Goods]
		goodsPriceFen := money.NewFen(goodsInfo.ShopPriceFen)
		lineAmountFen, err := goodsPriceFen.Multiply(int64(value.Nums))
		if err != nil {
			return errors.Wrap(err, "calculate order line amount")
		}
		orderAmountFen, err = orderAmountFen.Add(lineAmountFen)
		if err != nil {
			return errors.Wrap(err, "accumulate order amount")
		}
		value.GoodsName = goodsInfo.Name
		value.GoodsPriceFen = goodsPriceFen.Int64()
		value.GoodsImage = goodsInfo.GoodsFrontImage
	}
	order.OrderMountFen = orderAmountFen.Int64()

	txn := os.data.Begin() //开启事务
	if txn == nil {
		if err := os.data.Orders().Create(ctx, nil, &order.OrderInfoDO); err != nil {
			return fmt.Errorf("create order: %w", err)
		}
		if err := os.data.ShopCarts().DeleteByGoodsIDs(ctx, nil, uint64(order.User), goodsIds); err != nil {
			return fmt.Errorf("delete selected shop carts: %w", err)
		}
		if err := os.createStatusLog(ctx, nil, &do.OrderStatusLogDO{
			OrderID:    order.ID,
			OrderSn:    order.OrderSn,
			FromStatus: "",
			ToStatus:   order.Status,
			Reason:     "order created",
			Source:     "order.create",
			Operator:   fmt.Sprintf("user:%d", order.User),
		}); err != nil {
			return fmt.Errorf("create order status log: %w", err)
		}
		return nil
	}
	if txn.Error != nil {
		return fmt.Errorf("begin order transaction: %w", txn.Error)
	}
	rollback := func(cause error) error {
		if rbErr := txn.Rollback().Error; rbErr != nil {
			return fmt.Errorf("%w; rollback order transaction: %v", cause, rbErr)
		}
		return cause
	}
	defer func() {
		if panicValue := recover(); panicValue != nil {
			err = rollback(fmt.Errorf("create order transaction panic: %v", panicValue))
		}
	}()

	err = os.data.Orders().Create(ctx, txn, &order.OrderInfoDO)
	if err != nil {
		return rollback(fmt.Errorf("create order: %w", err))
	}

	err = os.data.ShopCarts().DeleteByGoodsIDs(ctx, txn, uint64(order.User), goodsIds)
	if err != nil {
		return rollback(fmt.Errorf("delete selected shop carts: %w", err))
	}

	if err := os.createStatusLog(ctx, txn, &do.OrderStatusLogDO{
		OrderID:    order.ID,
		OrderSn:    order.OrderSn,
		FromStatus: "",
		ToStatus:   order.Status,
		Reason:     "order created",
		Source:     "order.create",
		Operator:   fmt.Sprintf("user:%d", order.User),
	}); err != nil {
		return rollback(fmt.Errorf("create order status log: %w", err))
	}

	if err := txn.Commit().Error; err != nil {
		return fmt.Errorf("commit order transaction: %w", err)
	}
	return nil
}

func sameCreateOrder(existing *do.OrderInfoDO, incoming *dto.OrderDTO) bool {
	if existing == nil || incoming == nil {
		return false
	}
	if existing.User != incoming.User {
		return false
	}
	if strings.TrimSpace(existing.OrderSn) != strings.TrimSpace(incoming.OrderSn) {
		return false
	}
	if strings.TrimSpace(existing.Address) != strings.TrimSpace(incoming.Address) {
		return false
	}
	if strings.TrimSpace(existing.SignerName) != strings.TrimSpace(incoming.SignerName) {
		return false
	}
	if strings.TrimSpace(existing.SingerMobile) != strings.TrimSpace(incoming.SingerMobile) {
		return false
	}
	if strings.TrimSpace(existing.Post) != strings.TrimSpace(incoming.Post) {
		return false
	}
	return sameOrderGoods(existing.OrderGoods, incoming.OrderGoods)
}

func hasOrderGoods(items []*do.OrderGoods) bool {
	_, ok := aggregateOrderGoods(items)
	return ok
}

func sameOrderGoods(left, right []*do.OrderGoods) bool {
	leftGoods, ok := aggregateOrderGoods(left)
	if !ok {
		return false
	}
	rightGoods, ok := aggregateOrderGoods(right)
	if !ok {
		return false
	}
	if len(leftGoods) != len(rightGoods) {
		return false
	}
	for goodsID, nums := range leftGoods {
		if rightGoods[goodsID] != nums {
			return false
		}
	}
	return true
}

func aggregateOrderGoods(items []*do.OrderGoods) (map[int32]int64, bool) {
	if len(items) == 0 {
		return nil, false
	}
	goods := make(map[int32]int64, len(items))
	for _, item := range items {
		if item == nil || item.Goods <= 0 || item.Nums <= 0 {
			return nil, false
		}
		goods[item.Goods] += int64(item.Nums)
	}
	return goods, len(goods) > 0
}

func (os *orderService) Get(ctx context.Context, userID uint64, orderSn string) (*dto.OrderDTO, error) {
	if userID == 0 || strings.TrimSpace(orderSn) == "" {
		return nil, errors.WithCode(code.ErrOrderNotFound, "order not found")
	}

	order, err := os.data.Orders().Get(ctx, orderSn)
	if err != nil {
		return nil, err
	}
	if uint64(order.User) != userID {
		return nil, errors.WithCode(code.ErrOrderNotFound, "order not found")
	}
	return &dto.OrderDTO{OrderInfoDO: *order}, nil
}

func (os *orderService) StatusLogs(ctx context.Context, userID uint64, orderSn string) (*dto.OrderStatusLogDTOList, error) {
	order, err := os.Get(ctx, userID, orderSn)
	if err != nil {
		return nil, err
	}

	entries, err := os.data.OrderStatusLogs().ListByOrderSn(ctx, order.OrderSn)
	if err != nil {
		return nil, err
	}

	ret := &dto.OrderStatusLogDTOList{TotalCount: int64(len(entries))}
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		ret.Items = append(ret.Items, &dto.OrderStatusLogDTO{OrderStatusLogDO: *entry})
	}
	return ret, nil
}

func (os *orderService) List(ctx context.Context, userID uint64, meta v1.ListMeta, orderBy []string) (*dto.OrderDTOList, error) {
	if userID == 0 {
		return &dto.OrderDTOList{}, nil
	}

	orders, err := os.data.Orders().List(ctx, userID, meta, orderBy)
	if err != nil {
		return nil, err
	}
	var ret dto.OrderDTOList
	ret.TotalCount = orders.TotalCount
	for _, value := range orders.Items {
		ret.Items = append(ret.Items, &dto.OrderDTO{
			OrderInfoDO: *value,
		})
	}
	return &ret, nil
}

func (os *orderService) Submit(ctx context.Context, order *dto.OrderDTO) error {
	if order == nil {
		return errors.WithCode(code.ErrSubmitOrder, "order is required")
	}
	order.OrderSn = strings.TrimSpace(order.OrderSn)
	if order.User <= 0 || order.OrderSn == "" {
		return errors.WithCode(code.ErrSubmitOrder, "user and order_sn are required")
	}
	if os.dtmOpts == nil || strings.TrimSpace(os.dtmOpts.GrpcServer) == "" {
		return errors.WithCode(code.ErrSubmitOrder, "dtm grpc server is required")
	}

	//先从购物车中获取商品信息
	list, err := os.data.ShopCarts().List(ctx, uint64(order.User), true, v1.ListMeta{}, []string{})
	if err != nil {
		log.Errorf("获取购物车信息失败，err:%v", err)
		return err
	}

	if len(list.Items) == 0 {
		log.Errorf("购物车中没有商品，无法下单")
		return errors.WithCode(code.ErrNoGoodsSelect, "没有选择商品")
	}

	var orderGoods []*do.OrderGoods
	var orderItems []*proto3.OrderItemResponse
	for _, value := range list.Items {
		orderGoods = append(orderGoods, &do.OrderGoods{
			Goods: value.Goods,
			Nums:  value.Nums,
		})

		orderItems = append(orderItems, &proto3.OrderItemResponse{
			GoodsId: value.Goods,
			Nums:    value.Nums,
		})
	}
	order.OrderGoods = orderGoods

	//基于可靠消息最终一致性的思想， saga事务来解决订单生成的问题
	var goodsInfo []*proto.GoodsInvInfo
	for _, value := range order.OrderGoods {
		goodsInfo = append(goodsInfo, &proto.GoodsInvInfo{
			GoodsId: value.Goods,
			Num:     value.Nums,
		})
	}
	req := &proto.SellInfo{
		GoodsInfo: goodsInfo,
		OrderSn:   order.OrderSn,
	}
	oReq := &proto3.OrderRequest{
		OrderSn:    order.OrderSn,
		UserId:     order.User,
		Address:    order.Address,
		Name:       order.SignerName,
		Mobile:     order.SingerMobile,
		Post:       order.Post,
		OrderItems: orderItems,
	}

	saga := dtmgrpc.NewSagaGrpc(os.dtmOpts.GrpcServer, order.OrderSn).
		Add(inventory_busi+"/Inventory/Sell", inventory_busi+"/Inventory/Reback", req).
		Add(order_busi+"/Order/CreateOrder", order_busi+"/Order/CreateOrderCom", oReq)
	saga.WaitResult = true
	err = saga.Submit()
	//通过OrderSn查询一下， 当前的状态如何状态一直是Submitted那么就你一直不要给前端返回， 如果是failed那么你提示给前端说下单失败，重新下单
	return err
}

func (os *orderService) Update(ctx context.Context, order *dto.OrderDTO) error {
	if order == nil {
		return errors.WithCode(code.ErrOrderStatusInvalid, "order is required")
	}
	order.OrderSn = strings.TrimSpace(order.OrderSn)
	order.Status = strings.TrimSpace(order.Status)
	if order.OrderSn == "" || order.Status == "" {
		return errors.WithCode(code.ErrOrderStatusInvalid, "order_sn and status are required")
	}
	if !isValidOrderStatus(order.Status) {
		return errors.WithCode(code.ErrOrderStatusInvalid, "invalid order status")
	}

	existing, err := os.data.Orders().Get(ctx, order.OrderSn)
	if err != nil {
		return err
	}
	trigger := normalizeTransitionTrigger(defaultStatusLogSource(order.StatusSource, existing.Status, order.Status))
	transition := transitionMetricName(existing.Status, order.Status)
	result := "success"
	startedAt := time.Now()
	defer func() {
		observeOrderTransition(trigger, transition, result, startedAt)
	}()
	if !canTransitionOrderStatus(existing.Status, order.Status) {
		result = "invalid"
		return errors.WithCode(code.ErrOrderStatusInvalid, "invalid order status transition")
	}
	if order.Status == OrderStatusTradeSuccess {
		if strings.TrimSpace(order.TradeNo) == "" {
			result = "invalid"
			return errors.WithCode(code.ErrOrderStatusInvalid, "trade_no is required when order is paid")
		}
		if order.PayTime == nil {
			now := time.Now()
			order.PayTime = &now
		}
	}
	if order.Status == OrderStatusRefundPending {
		if order.ActorUserID <= 0 || order.RefundAmountFen <= 0 || strings.TrimSpace(order.StatusReason) == "" || strings.TrimSpace(order.CorrelationID) == "" {
			result = "invalid"
			return errors.WithCode(code.ErrOrderStatusInvalid, "refund actor, amount, reason, and correlation_id are required")
		}
		if order.RefundAmountFen > existing.OrderMountFen {
			result = "invalid"
			return errors.WithCode(code.ErrOrderStatusInvalid, "refund amount exceeds order total")
		}
	}

	txn := os.data.Begin()
	if txn == nil {
		if err := os.data.Orders().Update(ctx, nil, &order.OrderInfoDO); err != nil {
			result = "failed"
			return err
		}
		if strings.TrimSpace(existing.Status) != strings.TrimSpace(order.Status) {
			if err := os.createStatusLog(ctx, nil, os.buildStatusLog(existing, order)); err != nil {
				result = "failed"
				return fmt.Errorf("create order status log: %w", err)
			}
			logOrderTransitionSuccess(ctx, existing, order)
			return nil
		}
		result = "noop"
		return nil
	}
	if txn.Error != nil {
		result = "failed"
		return fmt.Errorf("begin update order transaction: %w", txn.Error)
	}

	if err := os.data.Orders().Update(ctx, txn, &order.OrderInfoDO); err != nil {
		result = "failed"
		_ = txn.Rollback()
		return err
	}

	if strings.TrimSpace(existing.Status) != strings.TrimSpace(order.Status) {
		if err := os.createStatusLog(ctx, txn, os.buildStatusLog(existing, order)); err != nil {
			result = "failed"
			_ = txn.Rollback()
			return fmt.Errorf("create order status log: %w", err)
		}
	}
	if order.Status == OrderStatusRefundPending {
		store, ok := os.data.Orders().(interface {
			CreateRefundRequest(context.Context, *gorm.DB, *do.RefundRequestDO) error
		})
		if !ok {
			_ = txn.Rollback()
			return errors.WithCode(code.ErrOrderStatusInvalid, "refund store is not configured")
		}
		now := time.Now().UTC()
		if err := store.CreateRefundRequest(ctx, txn, &do.RefundRequestDO{OrderSN: order.OrderSn, ActorUserID: order.ActorUserID, AmountFen: order.RefundAmountFen, Reason: order.StatusReason, Status: OrderStatusRefundPending, CorrelationID: order.CorrelationID, CreatedAt: now, UpdatedAt: now}); err != nil {
			_ = txn.Rollback()
			return err
		}
	}

	if err := txn.Commit().Error; err != nil {
		result = "failed"
		return fmt.Errorf("commit update order transaction: %w", err)
	}
	if strings.TrimSpace(existing.Status) != strings.TrimSpace(order.Status) {
		logOrderTransitionSuccess(ctx, existing, order)
	} else {
		result = "noop"
	}
	return nil
}

func isValidOrderStatus(status string) bool {
	_, ok := validOrderStatuses[status]
	return ok
}

func canTransitionOrderStatus(current, next string) bool {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if current == "" || current == next {
		return true
	}

	switch current {
	case OrderStatusWaitBuyerPay:
		return next == OrderStatusPaying || next == OrderStatusTradeSuccess || next == OrderStatusTradeClosed
	case OrderStatusPaying:
		return next == OrderStatusTradeSuccess || next == OrderStatusTradeClosed
	case OrderStatusTradeClosed, OrderStatusTradeFinished:
		return current == OrderStatusTradeFinished && next == OrderStatusRefundPending
	case OrderStatusTradeSuccess:
		return next == OrderStatusTradeFinished || next == OrderStatusRefundPending
	case OrderStatusRefundPending:
		return next == OrderStatusRefunded || next == OrderStatusRefundFailed
	case OrderStatusRefundFailed:
		return next == OrderStatusRefundPending
	case OrderStatusRefunded:
		return false
	default:
		return false
	}
}

func normalizeInitialOrderStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return OrderStatusWaitBuyerPay
	}
	return status
}

func newOrderService(sv *service) *orderService {
	return &orderService{
		data:     sv.data,
		dtmOpts:  sv.dtmopts,
		upstream: sv.upstream,
	}
}

var _ OrderSrv = &orderService{}

func (os *orderService) createStatusLog(ctx context.Context, txn *gorm.DB, entry *do.OrderStatusLogDO) error {
	if os == nil || os.data == nil || entry == nil {
		return nil
	}
	store := os.data.OrderStatusLogs()
	if store == nil {
		return nil
	}
	return store.Create(ctx, txn, entry)
}

func (os *orderService) buildStatusLog(existing *do.OrderInfoDO, order *dto.OrderDTO) *do.OrderStatusLogDO {
	if existing == nil || order == nil {
		return nil
	}
	return &do.OrderStatusLogDO{
		OrderID:    existing.ID,
		OrderSn:    existing.OrderSn,
		FromStatus: existing.Status,
		ToStatus:   order.Status,
		Reason:     defaultStatusLogReason(order.StatusReason, existing.Status, order.Status),
		Source:     defaultStatusLogSource(order.StatusSource, existing.Status, order.Status),
		Operator:   defaultStatusLogOperator(order.StatusOperator, existing.User),
	}
}

func defaultStatusLogReason(reason, fromStatus, toStatus string) string {
	reason = strings.TrimSpace(reason)
	if reason != "" {
		return reason
	}
	switch {
	case fromStatus == "" && toStatus == OrderStatusWaitBuyerPay:
		return "order created"
	case toStatus == OrderStatusTradeSuccess:
		return "payment confirmed"
	case toStatus == OrderStatusTradeClosed:
		return "order closed"
	case toStatus == OrderStatusTradeFinished:
		return "order finished"
	default:
		return "order status changed"
	}
}

func defaultStatusLogSource(source, fromStatus, toStatus string) string {
	source = strings.TrimSpace(source)
	if source != "" {
		return source
	}
	switch {
	case fromStatus == "" && toStatus == OrderStatusWaitBuyerPay:
		return "order.create"
	case toStatus == OrderStatusTradeSuccess:
		return "order.payment"
	case toStatus == OrderStatusTradeClosed:
		return "order.close"
	case toStatus == OrderStatusTradeFinished:
		return "order.finish"
	default:
		return "order.update"
	}
}

func defaultStatusLogOperator(operator string, userID int32) string {
	operator = strings.TrimSpace(operator)
	if operator != "" {
		return operator
	}
	if userID > 0 {
		return fmt.Sprintf("user:%d", userID)
	}
	return "system"
}

func logOrderTransitionSuccess(ctx context.Context, existing *do.OrderInfoDO, order *dto.OrderDTO) {
	if existing == nil || order == nil {
		return
	}

	log.InfoC(ctx, "order status transition applied",
		log.Int32(log.KeyOrderID, existing.ID),
		log.Int32(log.KeyUserID, existing.User),
		log.String(log.KeyOrderSN, existing.OrderSn),
		log.String(log.KeyFromStatus, strings.TrimSpace(existing.Status)),
		log.String(log.KeyToStatus, strings.TrimSpace(order.Status)),
		log.String(log.KeyReason, defaultStatusLogReason(order.StatusReason, existing.Status, order.Status)),
		log.String(log.KeySource, defaultStatusLogSource(order.StatusSource, existing.Status, order.Status)),
		log.String(log.KeyOperator, defaultStatusLogOperator(order.StatusOperator, existing.User)),
	)
}
