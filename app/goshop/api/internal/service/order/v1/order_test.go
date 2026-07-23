package v1

import (
	"context"
	stderrors "errors"
	"testing"

	gpb "goshop/api/goods/v1"
	ipb "goshop/api/inventory/v1"
	opb "goshop/api/order/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/goshop/api/internal/payment"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestSimulatePayCallbackRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		svc  OrderSrv
		req  *PayCallbackRequest
		code int
	}{
		{
			name: "missing data",
			svc:  NewOrderService(nil),
			req:  &PayCallbackRequest{UserID: 1, OrderSn: "order-1"},
			code: code.ErrConnectGRPC,
		},
		{
			name: "nil request",
			svc:  NewOrderService(fakeDataFactory{}),
			code: code.ErrOrderStatusInvalid,
		},
		{
			name: "missing user",
			svc:  NewOrderService(fakeDataFactory{}),
			req:  &PayCallbackRequest{OrderSn: "order-1"},
			code: code.ErrOrderStatusInvalid,
		},
		{
			name: "missing trade no for success",
			svc:  NewOrderService(fakeDataFactory{orderClient: fakeOrderClient{detail: orderDetailResponse}, inventoryClient: fakeInventoryClient{}}),
			req:  &PayCallbackRequest{UserID: 1, OrderSn: "order-1", Success: true},
			code: code.ErrOrderStatusInvalid,
		},
		{
			name: "empty order goods",
			svc: NewOrderService(fakeDataFactory{
				orderClient: fakeOrderClient{
					detail: func(context.Context, *opb.OrderRequest, ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
						return &opb.OrderInfoDetailResponse{
							OrderInfo: &opb.OrderInfoResponse{OrderSn: "order-1", Status: orderStatusWaitBuyerPay},
						}, nil
					},
				},
				inventoryClient: fakeInventoryClient{},
			}),
			req:  &PayCallbackRequest{UserID: 1, OrderSn: "order-1", TradeNo: "trade-1", Success: true},
			code: code.ErrOrderStatusInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.svc.SimulatePayCallback(context.Background(), tt.req)
			if !errors.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want code %d", err, tt.code)
			}
		})
	}
}

func TestSimulatePayCallbackWritesExpectedStatus(t *testing.T) {
	var gotDetail *opb.OrderRequest
	var got *opb.OrderStatus
	var confirm *ipb.SellInfo
	var calls []string
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			detail: func(_ context.Context, in *opb.OrderRequest, _ ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
				gotDetail = in
				calls = append(calls, "detail")
				return orderDetailResponse(nil, nil)
			},
			update: func(_ context.Context, in *opb.OrderStatus, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				calls = append(calls, "update")
				got = in
				return &emptypb.Empty{}, nil
			},
		},
		inventoryClient: fakeInventoryClient{
			confirm: func(_ context.Context, in *ipb.SellInfo, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				calls = append(calls, "confirm")
				confirm = in
				return &emptypb.Empty{}, nil
			},
		},
	})

	err := svc.SimulatePayCallback(context.Background(), &PayCallbackRequest{
		UserID:  9,
		OrderSn: " order-9 ",
		PayType: "alipay",
		TradeNo: "trade-1",
		Success: true,
	})
	if err != nil {
		t.Fatalf("SimulatePayCallback() error = %v", err)
	}
	if gotDetail == nil || gotDetail.UserId != 9 || gotDetail.OrderSn != "order-9" {
		t.Fatalf("OrderDetail() got %+v", gotDetail)
	}
	if got == nil {
		t.Fatal("SimulatePayCallback() did not call UpdateOrderStatus")
	}
	if got.OrderSn != "order-9" || got.Status != "TRADE_SUCCESS" || got.PayType != "alipay" || got.TradeNo != "trade-1" || got.PayTime <= 0 {
		t.Fatalf("UpdateOrderStatus() got %+v", got)
	}
	if confirm == nil || confirm.OrderSn != "order-9" || len(confirm.GoodsInfo) != 1 || confirm.GoodsInfo[0].GoodsId != 101 || confirm.GoodsInfo[0].Num != 2 {
		t.Fatalf("Confirm() got %+v", confirm)
	}
	if len(calls) != 3 || calls[0] != "detail" || calls[1] != "confirm" || calls[2] != "update" {
		t.Fatalf("call order = %v, want [detail confirm update]", calls)
	}
}

func TestSimulatePayCallbackCanCloseOrder(t *testing.T) {
	var got *opb.OrderStatus
	var release *ipb.SellInfo
	var calls []string
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			detail: func(_ context.Context, _ *opb.OrderRequest, _ ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
				calls = append(calls, "detail")
				return orderDetailResponse(nil, nil)
			},
			update: func(_ context.Context, in *opb.OrderStatus, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				calls = append(calls, "update")
				got = in
				return &emptypb.Empty{}, nil
			},
		},
		inventoryClient: fakeInventoryClient{
			release: func(_ context.Context, in *ipb.SellInfo, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				calls = append(calls, "release")
				release = in
				return &emptypb.Empty{}, nil
			},
		},
	})

	err := svc.SimulatePayCallback(context.Background(), &PayCallbackRequest{
		UserID:  9,
		OrderSn: "order-9",
		Success: false,
	})
	if err != nil {
		t.Fatalf("SimulatePayCallback() error = %v", err)
	}
	if got == nil || got.Status != "TRADE_CLOSED" {
		t.Fatalf("UpdateOrderStatus() got %+v, want TRADE_CLOSED", got)
	}
	if release == nil || release.OrderSn != "order-9" || len(release.GoodsInfo) != 1 || release.GoodsInfo[0].GoodsId != 101 || release.GoodsInfo[0].Num != 2 {
		t.Fatalf("Release() got %+v", release)
	}
	if len(calls) != 3 || calls[0] != "detail" || calls[1] != "release" || calls[2] != "update" {
		t.Fatalf("call order = %v, want [detail release update]", calls)
	}
}

func TestSimulatePayCallbackRejectsConflictingTerminalState(t *testing.T) {
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			detail: func(_ context.Context, _ *opb.OrderRequest, _ ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
				return &opb.OrderInfoDetailResponse{
					OrderInfo: &opb.OrderInfoResponse{OrderSn: "order-9", Status: orderStatusTradeClosed},
					Goods: []*opb.OrderItemResponse{
						{GoodsId: 101, Nums: 2},
					},
				}, nil
			},
		},
		inventoryClient: fakeInventoryClient{},
	})

	err := svc.SimulatePayCallback(context.Background(), &PayCallbackRequest{
		UserID:  9,
		OrderSn: "order-9",
		TradeNo: "trade-1",
		Success: true,
	})
	if !errors.IsCode(err, code.ErrOrderStatusInvalid) {
		t.Fatalf("SimulatePayCallback() error = %v, want code %d", err, code.ErrOrderStatusInvalid)
	}
}

func TestProcessPayCallbackCompletedReplayHasNoSideEffects(t *testing.T) {
	mutations := 0
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{beginPaymentEvent: func(context.Context, *opb.PaymentEventRequest, ...grpc.CallOption) (*opb.PaymentEventResponse, error) {
			return &opb.PaymentEventResponse{Id: 1, Completed: true}, nil
		}, update: func(context.Context, *opb.OrderStatus, ...grpc.CallOption) (*emptypb.Empty, error) {
			mutations++
			return &emptypb.Empty{}, nil
		}},
		inventoryClient: fakeInventoryClient{confirm: func(context.Context, *ipb.SellInfo, ...grpc.CallOption) (*emptypb.Empty, error) {
			mutations++
			return &emptypb.Empty{}, nil
		}},
	}).(*orderService)

	duplicate, err := svc.ProcessPayCallback(context.Background(), &payment.CallbackRequest{Provider: "mock", EventID: "evt-1", EventType: "payment_succeeded", OrderSN: "order-1", AmountFen: 100})
	if err != nil || !duplicate {
		t.Fatalf("ProcessPayCallback() duplicate=%v error=%v", duplicate, err)
	}
	if mutations != 0 {
		t.Fatalf("replayed callback caused %d mutations", mutations)
	}
}

func TestProcessPayCallbackRejectsOutOfOrderBeforeInventory(t *testing.T) {
	inventoryCalls, completeCalls := 0, 0
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			beginPaymentEvent: func(context.Context, *opb.PaymentEventRequest, ...grpc.CallOption) (*opb.PaymentEventResponse, error) {
				return &opb.PaymentEventResponse{Id: 1, Accepted: true, OrderAmountFen: 100}, nil
			},
			getOrderBySN: func(context.Context, *opb.OrderLookupRequest, ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
				return orderDetailResponseWithStatus(orderStatusTradeClosed), nil
			},
			completePaymentEvent: func(_ context.Context, in *opb.CompletePaymentEventRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				completeCalls++
				if in.Success || in.ErrorDetail != "payment event is out of order" {
					t.Fatalf("completion = %+v", in)
				}
				return &emptypb.Empty{}, nil
			},
		},
		inventoryClient: fakeInventoryClient{confirm: func(context.Context, *ipb.SellInfo, ...grpc.CallOption) (*emptypb.Empty, error) {
			inventoryCalls++
			return &emptypb.Empty{}, nil
		}},
	}).(*orderService)

	_, err := svc.ProcessPayCallback(context.Background(), &payment.CallbackRequest{Provider: "mock", EventID: "evt-2", EventType: "payment_succeeded", OrderSN: "order-1", AmountFen: 100})
	if !errors.IsCode(err, code.ErrOrderStatusInvalid) {
		t.Fatalf("ProcessPayCallback() error=%v", err)
	}
	if inventoryCalls != 0 || completeCalls != 1 {
		t.Fatalf("inventoryCalls=%d completeCalls=%d", inventoryCalls, completeCalls)
	}
}

func orderDetailResponseWithStatus(status string) *opb.OrderInfoDetailResponse {
	return &opb.OrderInfoDetailResponse{
		OrderInfo: &opb.OrderInfoResponse{OrderSn: "order-1", Status: status, TotalFen: 100},
		Goods:     []*opb.OrderItemResponse{{GoodsId: 101, Nums: 1}},
	}
}

func TestCartItemOperationsForwardRequests(t *testing.T) {
	var gotCreate *opb.CartItemRequest
	var gotUpdate *opb.CartItemRequest
	var gotDelete *opb.CartItemRequest
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			createCartItem: func(_ context.Context, in *opb.CartItemRequest, _ ...grpc.CallOption) (*opb.ShopCartInfoResponse, error) {
				gotCreate = in
				return &opb.ShopCartInfoResponse{
					Id:      1,
					UserId:  in.UserId,
					GoodsId: in.GoodsId,
					Nums:    in.Nums,
					Checked: in.Checked,
				}, nil
			},
			updateCartItem: func(_ context.Context, in *opb.CartItemRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				gotUpdate = in
				return &emptypb.Empty{}, nil
			},
			deleteCartItem: func(_ context.Context, in *opb.CartItemRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				gotDelete = in
				return &emptypb.Empty{}, nil
			},
		},
	})

	item, err := svc.CreateCartItem(context.Background(), 9, &CartItemRequest{GoodsID: 101, Nums: 2, Checked: true})
	if err != nil {
		t.Fatalf("CreateCartItem() error = %v", err)
	}
	if gotCreate == nil || gotCreate.UserId != 9 || gotCreate.GoodsId != 101 || gotCreate.Nums != 2 || !gotCreate.Checked {
		t.Fatalf("CreateCartItem() request = %+v", gotCreate)
	}
	if item == nil || item.Id != 1 {
		t.Fatalf("CreateCartItem() response = %+v", item)
	}

	if err := svc.UpdateCartItem(context.Background(), 9, &CartItemRequest{GoodsID: 101, Nums: 3, Checked: false}); err != nil {
		t.Fatalf("UpdateCartItem() error = %v", err)
	}
	if gotUpdate == nil || gotUpdate.UserId != 9 || gotUpdate.GoodsId != 101 || gotUpdate.Nums != 3 || gotUpdate.Checked {
		t.Fatalf("UpdateCartItem() request = %+v", gotUpdate)
	}

	if err := svc.DeleteCartItem(context.Background(), 9, 7); err != nil {
		t.Fatalf("DeleteCartItem() error = %v", err)
	}
	if gotDelete == nil || gotDelete.UserId != 9 || gotDelete.Id != 7 {
		t.Fatalf("DeleteCartItem() request = %+v", gotDelete)
	}
}

func TestSubmitOrderGeneratesOrderSnAndForwardsFields(t *testing.T) {
	var got *opb.OrderRequest
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			submitOrder: func(_ context.Context, in *opb.OrderRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
				got = in
				return &emptypb.Empty{}, nil
			},
		},
	})

	orderSn, err := svc.SubmitOrder(context.Background(), 9, &SubmitOrderRequest{
		Address: "上海市浦东新区",
		Name:    "buyer",
		Mobile:  "13800138000",
		Post:    "尽快发货",
	})
	if err != nil {
		t.Fatalf("SubmitOrder() error = %v", err)
	}
	if orderSn == "" {
		t.Fatal("SubmitOrder() orderSn is empty")
	}
	if got == nil || got.UserId != 9 || got.OrderSn != orderSn || got.Address != "上海市浦东新区" || got.Name != "buyer" || got.Mobile != "13800138000" || got.Post != "尽快发货" {
		t.Fatalf("SubmitOrder() request = %+v", got)
	}
}

func TestOrderQueriesForwardRequests(t *testing.T) {
	var gotCartList *opb.UserInfo
	var gotList *opb.OrderFilterRequest
	var gotDetail *opb.OrderRequest
	var gotLogs *opb.OrderRequest
	svc := NewOrderService(fakeDataFactory{
		orderClient: fakeOrderClient{
			cartItemList: func(_ context.Context, in *opb.UserInfo, _ ...grpc.CallOption) (*opb.CartItemListResponse, error) {
				gotCartList = in
				return &opb.CartItemListResponse{
					Total: 1,
					Data: []*opb.ShopCartInfoResponse{
						{Id: 1, UserId: in.Id, GoodsId: 101, Nums: 2, Checked: true},
					},
				}, nil
			},
			list: func(_ context.Context, in *opb.OrderFilterRequest, _ ...grpc.CallOption) (*opb.OrderListResponse, error) {
				gotList = in
				return &opb.OrderListResponse{
					Total: 1,
					Data: []*opb.OrderInfoResponse{
						{OrderSn: "order-1", UserId: in.UserId},
					},
				}, nil
			},
			detail: func(_ context.Context, in *opb.OrderRequest, _ ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
				gotDetail = in
				return orderDetailResponse(nil, nil)
			},
			statusLogs: func(_ context.Context, in *opb.OrderRequest, _ ...grpc.CallOption) (*opb.OrderStatusLogListResponse, error) {
				gotLogs = in
				return &opb.OrderStatusLogListResponse{
					Total: 1,
					Data: []*opb.OrderStatusLogResponse{
						{OrderSn: in.OrderSn, ToStatus: orderStatusTradeSuccess},
					},
				}, nil
			},
		},
	})

	cartResp, err := svc.CartItemList(context.Background(), 9)
	if err != nil {
		t.Fatalf("CartItemList() error = %v", err)
	}
	if gotCartList == nil || gotCartList.Id != 9 || cartResp.GetTotal() != 1 {
		t.Fatalf("CartItemList() request=%+v response=%+v", gotCartList, cartResp)
	}

	listResp, err := svc.OrderList(context.Background(), 9, &OrderListFilter{})
	if err != nil {
		t.Fatalf("OrderList() error = %v", err)
	}
	if gotList == nil || gotList.UserId != 9 || gotList.Pages != defaultOrderPage || gotList.PagePerNums != defaultOrderPageSize || listResp.GetTotal() != 1 {
		t.Fatalf("OrderList() request=%+v response=%+v", gotList, listResp)
	}

	detailResp, err := svc.OrderDetail(context.Background(), 9, " order-1 ")
	if err != nil {
		t.Fatalf("OrderDetail() error = %v", err)
	}
	if gotDetail == nil || gotDetail.UserId != 9 || gotDetail.OrderSn != "order-1" || detailResp.GetOrderInfo().GetOrderSn() != "order-9" {
		t.Fatalf("OrderDetail() request=%+v response=%+v", gotDetail, detailResp)
	}

	logResp, err := svc.OrderStatusLogs(context.Background(), 9, " order-1 ")
	if err != nil {
		t.Fatalf("OrderStatusLogs() error = %v", err)
	}
	if gotLogs == nil || gotLogs.UserId != 9 || gotLogs.OrderSn != "order-1" || logResp.GetTotal() != 1 || len(logResp.GetData()) != 1 {
		t.Fatalf("OrderStatusLogs() request=%+v response=%+v", gotLogs, logResp)
	}
}

type fakeDataFactory struct {
	inventoryClient ipb.InventoryClient
	orderClient     opb.OrderClient
}

func (f fakeDataFactory) Goods() gpb.GoodsClient {
	return nil
}

func (f fakeDataFactory) Orders() opb.OrderClient {
	return f.orderClient
}

func (f fakeDataFactory) Inventory() ipb.InventoryClient {
	return f.inventoryClient
}

func (f fakeDataFactory) Users() data.UserData {
	return nil
}

type fakeOrderClient struct {
	opb.OrderClient
	cartItemList         func(context.Context, *opb.UserInfo, ...grpc.CallOption) (*opb.CartItemListResponse, error)
	createCartItem       func(context.Context, *opb.CartItemRequest, ...grpc.CallOption) (*opb.ShopCartInfoResponse, error)
	updateCartItem       func(context.Context, *opb.CartItemRequest, ...grpc.CallOption) (*emptypb.Empty, error)
	deleteCartItem       func(context.Context, *opb.CartItemRequest, ...grpc.CallOption) (*emptypb.Empty, error)
	submitOrder          func(context.Context, *opb.OrderRequest, ...grpc.CallOption) (*emptypb.Empty, error)
	list                 func(context.Context, *opb.OrderFilterRequest, ...grpc.CallOption) (*opb.OrderListResponse, error)
	detail               func(context.Context, *opb.OrderRequest, ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error)
	statusLogs           func(context.Context, *opb.OrderRequest, ...grpc.CallOption) (*opb.OrderStatusLogListResponse, error)
	update               func(context.Context, *opb.OrderStatus, ...grpc.CallOption) (*emptypb.Empty, error)
	beginPaymentEvent    func(context.Context, *opb.PaymentEventRequest, ...grpc.CallOption) (*opb.PaymentEventResponse, error)
	completePaymentEvent func(context.Context, *opb.CompletePaymentEventRequest, ...grpc.CallOption) (*emptypb.Empty, error)
	getOrderBySN         func(context.Context, *opb.OrderLookupRequest, ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error)
}

func (f fakeOrderClient) BeginPaymentEvent(ctx context.Context, in *opb.PaymentEventRequest, opts ...grpc.CallOption) (*opb.PaymentEventResponse, error) {
	return f.beginPaymentEvent(ctx, in, opts...)
}

func (f fakeOrderClient) CompletePaymentEvent(ctx context.Context, in *opb.CompletePaymentEventRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return f.completePaymentEvent(ctx, in, opts...)
}

func (f fakeOrderClient) GetOrderBySn(ctx context.Context, in *opb.OrderLookupRequest, opts ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
	return f.getOrderBySN(ctx, in, opts...)
}

func (f fakeOrderClient) CartItemList(ctx context.Context, in *opb.UserInfo, opts ...grpc.CallOption) (*opb.CartItemListResponse, error) {
	if f.cartItemList != nil {
		return f.cartItemList(ctx, in, opts...)
	}
	return &opb.CartItemListResponse{}, nil
}

func (f fakeOrderClient) CreateCartItem(ctx context.Context, in *opb.CartItemRequest, opts ...grpc.CallOption) (*opb.ShopCartInfoResponse, error) {
	if f.createCartItem != nil {
		return f.createCartItem(ctx, in, opts...)
	}
	return &opb.ShopCartInfoResponse{}, nil
}

func (f fakeOrderClient) UpdateCartItem(ctx context.Context, in *opb.CartItemRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.updateCartItem != nil {
		return f.updateCartItem(ctx, in, opts...)
	}
	return &emptypb.Empty{}, nil
}

func (f fakeOrderClient) DeleteCartItem(ctx context.Context, in *opb.CartItemRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.deleteCartItem != nil {
		return f.deleteCartItem(ctx, in, opts...)
	}
	return &emptypb.Empty{}, nil
}

func (f fakeOrderClient) SubmitOrder(ctx context.Context, in *opb.OrderRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.submitOrder != nil {
		return f.submitOrder(ctx, in, opts...)
	}
	return &emptypb.Empty{}, nil
}

func (f fakeOrderClient) OrderList(ctx context.Context, in *opb.OrderFilterRequest, opts ...grpc.CallOption) (*opb.OrderListResponse, error) {
	if f.list != nil {
		return f.list(ctx, in, opts...)
	}
	return &opb.OrderListResponse{}, nil
}

func (f fakeOrderClient) OrderDetail(ctx context.Context, in *opb.OrderRequest, opts ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
	if f.detail != nil {
		return f.detail(ctx, in, opts...)
	}
	return orderDetailResponse(ctx, in, opts...)
}

func (f fakeOrderClient) OrderStatusLogs(ctx context.Context, in *opb.OrderRequest, opts ...grpc.CallOption) (*opb.OrderStatusLogListResponse, error) {
	if f.statusLogs != nil {
		return f.statusLogs(ctx, in, opts...)
	}
	return &opb.OrderStatusLogListResponse{}, nil
}

func (f fakeOrderClient) UpdateOrderStatus(ctx context.Context, in *opb.OrderStatus, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.update != nil {
		return f.update(ctx, in, opts...)
	}
	return &emptypb.Empty{}, nil
}

type fakeInventoryClient struct {
	ipb.InventoryClient
	confirm func(context.Context, *ipb.SellInfo, ...grpc.CallOption) (*emptypb.Empty, error)
	release func(context.Context, *ipb.SellInfo, ...grpc.CallOption) (*emptypb.Empty, error)
}

func (f fakeInventoryClient) Confirm(ctx context.Context, in *ipb.SellInfo, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.confirm != nil {
		return f.confirm(ctx, in, opts...)
	}
	return &emptypb.Empty{}, nil
}

func (f fakeInventoryClient) Release(ctx context.Context, in *ipb.SellInfo, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if f.release != nil {
		return f.release(ctx, in, opts...)
	}
	return &emptypb.Empty{}, nil
}

func orderDetailResponse(_ context.Context, _ *opb.OrderRequest, _ ...grpc.CallOption) (*opb.OrderInfoDetailResponse, error) {
	return &opb.OrderInfoDetailResponse{
		OrderInfo: &opb.OrderInfoResponse{
			OrderSn: "order-9",
			Status:  orderStatusWaitBuyerPay,
		},
		Goods: []*opb.OrderItemResponse{
			{GoodsId: 101, Nums: 2},
		},
	}, nil
}

func TestPayCallbackMetricResult(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "success", want: "success"},
		{name: "rejected invalid", err: errors.WithCode(code.ErrOrderStatusInvalid, "invalid"), want: "rejected"},
		{name: "rejected missing order", err: errors.WithCode(code.ErrOrderNotFound, "missing"), want: "rejected"},
		{name: "dependency", err: errors.WithCode(code.ErrConnectGRPC, "connect"), want: "dependency_error"},
		{name: "failed", err: stderrors.New("boom"), want: "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := payCallbackMetricResult(tt.err); got != tt.want {
				t.Fatalf("payCallbackMetricResult(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}
