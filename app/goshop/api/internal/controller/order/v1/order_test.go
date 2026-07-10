package order

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	opb "goshop/api/order/v1"
	"goshop/app/goshop/api/internal/service"
	goodsv1 "goshop/app/goshop/api/internal/service/goods/v1"
	inventoryv1 "goshop/app/goshop/api/internal/service/inventory/v1"
	orderv1 "goshop/app/goshop/api/internal/service/order/v1"
	smsv1 "goshop/app/goshop/api/internal/service/sms/v1"
	userv1 "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/pkg/code"
	"goshop/gmicro/server/restserver/middlewares"

	"github.com/gin-gonic/gin"
)

func TestOrderControllerRejectsMissingServiceFactory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := NewOrderController(nil, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/order/pay/callback", strings.NewReader(`{"order_sn":"o1","items":[{"goods_id":1,"num":1}],"success":true}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	controller.SimulatePayCallback(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertOrderErrorCode(t, recorder.Body.Bytes(), code.ErrConnectGRPC)
}

func TestOrderControllerRejectsMissingUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := NewOrderController(&fakeOrderServiceFactory{orders: &fakeOrderSrv{}}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/order/pay/callback", strings.NewReader(`{"order_sn":"o1","items":[{"goods_id":1,"num":1}],"success":true}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	controller.SimulatePayCallback(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestOrderControllerSimulatesPayCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var got *orderv1.PayCallbackRequest
	controller := NewOrderController(&fakeOrderServiceFactory{
		orders: &fakeOrderSrv{
			simulate: func(_ context.Context, req *orderv1.PayCallbackRequest) error {
				got = req
				return nil
			},
		},
	}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middlewares.KeyUserID, float64(7))
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/order/pay/callback", strings.NewReader(`{"order_sn":"o1","pay_type":"wechat","trade_no":"t1","success":true}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	controller.SimulatePayCallback(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got == nil || got.UserID != 7 || got.OrderSn != "o1" || got.PayType != "wechat" || got.TradeNo != "t1" || !got.Success {
		t.Fatalf("request = %+v", got)
	}
	if len(got.Items) != 0 {
		t.Fatalf("items = %+v, want empty slice", got.Items)
	}
}

func TestOrderControllerListsCartItems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotUserID uint64
	controller := NewOrderController(&fakeOrderServiceFactory{
		orders: &fakeOrderSrv{
			listCartItems: func(_ context.Context, userID uint64) (*opb.CartItemListResponse, error) {
				gotUserID = userID
				return &opb.CartItemListResponse{
					Total: 1,
					Data: []*opb.ShopCartInfoResponse{
						{Id: 1, UserId: int32(userID), GoodsId: 101, Nums: 2, Checked: true},
					},
				}, nil
			},
		},
	}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middlewares.KeyUserID, float64(7))
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/user/cart_items", nil)

	controller.ListCartItems(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if gotUserID != 7 {
		t.Fatalf("userID = %d, want 7", gotUserID)
	}
	body, err := decodeJSONBody(recorder.Body.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["total"] != float64(1) {
		t.Fatalf("response body = %+v", body)
	}
	data, ok := body["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("response data = %+v", body["data"])
	}
}

func TestOrderControllerSubmitsOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotReq *orderv1.SubmitOrderRequest
	controller := NewOrderController(&fakeOrderServiceFactory{
		orders: &fakeOrderSrv{
			submitOrder: func(_ context.Context, userID uint64, req *orderv1.SubmitOrderRequest) (string, error) {
				if userID != 7 {
					t.Fatalf("userID = %d, want 7", userID)
				}
				gotReq = req
				return "order-1", nil
			},
		},
	}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middlewares.KeyUserID, float64(7))
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/user/orders", strings.NewReader(`{"address":"上海","name":"buyer","mobile":"13800138000","post":"尽快发货"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	controller.SubmitOrder(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if gotReq == nil || gotReq.Address != "上海" || gotReq.Name != "buyer" || gotReq.Mobile != "13800138000" || gotReq.Post != "尽快发货" {
		t.Fatalf("request = %+v", gotReq)
	}
	body, err := decodeJSONBody(recorder.Body.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["order_sn"] != "order-1" {
		t.Fatalf("response body = %+v", body)
	}
}

func TestOrderControllerReturnsOrderDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotOrderSn string
	controller := NewOrderController(&fakeOrderServiceFactory{
		orders: &fakeOrderSrv{
			orderDetail: func(_ context.Context, userID uint64, orderSn string) (*opb.OrderInfoDetailResponse, error) {
				if userID != 7 {
					t.Fatalf("userID = %d, want 7", userID)
				}
				gotOrderSn = orderSn
				return &opb.OrderInfoDetailResponse{
					OrderInfo: &opb.OrderInfoResponse{
						OrderSn: "order-1",
						Status:  "WAIT_BUYER_PAY",
					},
					Goods: []*opb.OrderItemResponse{
						{GoodsId: 101, Nums: 2},
					},
				}, nil
			},
		},
	}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middlewares.KeyUserID, float64(7))
	ctx.Params = gin.Params{{Key: "order_sn", Value: "order-1"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/user/orders/order-1", nil)

	controller.OrderDetail(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if gotOrderSn != "order-1" {
		t.Fatalf("orderSn = %s, want order-1", gotOrderSn)
	}
	body, err := decodeJSONBody(recorder.Body.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	orderInfo, ok := body["order_info"].(map[string]any)
	if !ok || orderInfo["order_sn"] != "order-1" || orderInfo["status"] != "WAIT_BUYER_PAY" {
		t.Fatalf("response body = %+v", body)
	}
}

func TestOrderControllerReturnsOrderStatusLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotOrderSn string
	controller := NewOrderController(&fakeOrderServiceFactory{
		orders: &fakeOrderSrv{
			orderStatusLogs: func(_ context.Context, userID uint64, orderSn string) (*opb.OrderStatusLogListResponse, error) {
				if userID != 7 {
					t.Fatalf("userID = %d, want 7", userID)
				}
				gotOrderSn = orderSn
				return &opb.OrderStatusLogListResponse{
					Total: 1,
					Data: []*opb.OrderStatusLogResponse{
						{
							OrderSn:    "order-1",
							FromStatus: "WAIT_BUYER_PAY",
							ToStatus:   "TRADE_SUCCESS",
							Reason:     "payment callback",
						},
					},
				}, nil
			},
		},
	}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(middlewares.KeyUserID, float64(7))
	ctx.Params = gin.Params{{Key: "order_sn", Value: "order-1"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/user/orders/order-1/status_logs", nil)

	controller.OrderStatusLogs(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if gotOrderSn != "order-1" {
		t.Fatalf("orderSn = %s, want order-1", gotOrderSn)
	}
	body, err := decodeJSONBody(recorder.Body.Bytes())
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["total"] != float64(1) {
		t.Fatalf("response body = %+v", body)
	}
	data, ok := body["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("response data = %+v", body["data"])
	}
}

func assertOrderErrorCode(t *testing.T, body []byte, want int) {
	t.Helper()
	var got struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if got.Code != want {
		t.Fatalf("code = %d, want %d", got.Code, want)
	}
}

type fakeOrderServiceFactory struct {
	orders orderv1.OrderSrv
}

func (f *fakeOrderServiceFactory) Goods() goodsv1.GoodsSrv {
	return nil
}

func (f *fakeOrderServiceFactory) Orders() orderv1.OrderSrv {
	return f.orders
}

func (f *fakeOrderServiceFactory) Inventory() inventoryv1.InventorySrv {
	return nil
}

func (f *fakeOrderServiceFactory) Users() userv1.UserSrv {
	return nil
}

func (f *fakeOrderServiceFactory) Sms() smsv1.SmsSrv {
	return nil
}

var _ service.ServiceFactory = &fakeOrderServiceFactory{}

type fakeOrderSrv struct {
	listCartItems   func(context.Context, uint64) (*opb.CartItemListResponse, error)
	createCartItem  func(context.Context, uint64, *orderv1.CartItemRequest) (*opb.ShopCartInfoResponse, error)
	updateCartItem  func(context.Context, uint64, *orderv1.CartItemRequest) error
	deleteCartItem  func(context.Context, uint64, uint64) error
	submitOrder     func(context.Context, uint64, *orderv1.SubmitOrderRequest) (string, error)
	orderList       func(context.Context, uint64, *orderv1.OrderListFilter) (*opb.OrderListResponse, error)
	orderDetail     func(context.Context, uint64, string) (*opb.OrderInfoDetailResponse, error)
	orderStatusLogs func(context.Context, uint64, string) (*opb.OrderStatusLogListResponse, error)
	simulate        func(context.Context, *orderv1.PayCallbackRequest) error
}

func (f *fakeOrderSrv) CartItemList(ctx context.Context, userID uint64) (*opb.CartItemListResponse, error) {
	if f.listCartItems != nil {
		return f.listCartItems(ctx, userID)
	}
	return &opb.CartItemListResponse{}, nil
}

func (f *fakeOrderSrv) CreateCartItem(ctx context.Context, userID uint64, req *orderv1.CartItemRequest) (*opb.ShopCartInfoResponse, error) {
	if f.createCartItem != nil {
		return f.createCartItem(ctx, userID, req)
	}
	return &opb.ShopCartInfoResponse{}, nil
}

func (f *fakeOrderSrv) UpdateCartItem(ctx context.Context, userID uint64, req *orderv1.CartItemRequest) error {
	if f.updateCartItem != nil {
		return f.updateCartItem(ctx, userID, req)
	}
	return nil
}

func (f *fakeOrderSrv) DeleteCartItem(ctx context.Context, userID, id uint64) error {
	if f.deleteCartItem != nil {
		return f.deleteCartItem(ctx, userID, id)
	}
	return nil
}

func (f *fakeOrderSrv) SubmitOrder(ctx context.Context, userID uint64, req *orderv1.SubmitOrderRequest) (string, error) {
	if f.submitOrder != nil {
		return f.submitOrder(ctx, userID, req)
	}
	return "", nil
}

func (f *fakeOrderSrv) OrderList(ctx context.Context, userID uint64, filter *orderv1.OrderListFilter) (*opb.OrderListResponse, error) {
	if f.orderList != nil {
		return f.orderList(ctx, userID, filter)
	}
	return &opb.OrderListResponse{}, nil
}

func (f *fakeOrderSrv) OrderDetail(ctx context.Context, userID uint64, orderSn string) (*opb.OrderInfoDetailResponse, error) {
	if f.orderDetail != nil {
		return f.orderDetail(ctx, userID, orderSn)
	}
	return &opb.OrderInfoDetailResponse{}, nil
}

func (f *fakeOrderSrv) OrderStatusLogs(ctx context.Context, userID uint64, orderSn string) (*opb.OrderStatusLogListResponse, error) {
	if f.orderStatusLogs != nil {
		return f.orderStatusLogs(ctx, userID, orderSn)
	}
	return &opb.OrderStatusLogListResponse{}, nil
}

func (f *fakeOrderSrv) SimulatePayCallback(ctx context.Context, req *orderv1.PayCallbackRequest) error {
	if f.simulate != nil {
		return f.simulate(ctx, req)
	}
	return nil
}
