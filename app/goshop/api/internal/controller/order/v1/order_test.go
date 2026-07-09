package order

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	simulate func(context.Context, *orderv1.PayCallbackRequest) error
}

func (f *fakeOrderSrv) SimulatePayCallback(ctx context.Context, req *orderv1.PayCallbackRequest) error {
	if f.simulate != nil {
		return f.simulate(ctx, req)
	}
	return nil
}
