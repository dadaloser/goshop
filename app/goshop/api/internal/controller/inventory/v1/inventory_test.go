package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ipb "goshop/api/inventory/v1"
	"goshop/app/goshop/api/internal/service"
	goodsv1 "goshop/app/goshop/api/internal/service/goods/v1"
	inventorysvc "goshop/app/goshop/api/internal/service/inventory/v1"
	orderv1 "goshop/app/goshop/api/internal/service/order/v1"
	smsv1 "goshop/app/goshop/api/internal/service/sms/v1"
	userv1 "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/pkg/code"

	"github.com/gin-gonic/gin"
)

func TestInventoryControllerRejectsMissingServiceFactory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := NewInventoryController(nil, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "goods_id", Value: "11"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/inventory/11", nil)

	controller.Detail(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertInventoryErrorCode(t, recorder.Body.Bytes(), code.ErrConnectGRPC)
}

func TestInventoryControllerRejectsInvalidGoodsID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := NewInventoryController(&fakeInventoryServiceFactory{}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "goods_id", Value: "0"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/inventory/0", nil)

	controller.Detail(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestInventoryControllerReturnsInventoryDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotGoodsID uint64
	controller := NewInventoryController(&fakeInventoryServiceFactory{
		inventory: fakeInventorySrv{
			detail: func(_ context.Context, goodsID uint64) (*ipb.GoodsInvInfo, error) {
				gotGoodsID = goodsID
				return &ipb.GoodsInvInfo{
					GoodsId:   11,
					Num:       7,
					Total:     10,
					Available: 7,
					Locked:    2,
					Sold:      1,
				}, nil
			},
		},
	}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "goods_id", Value: "11"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/inventory/11", nil)

	controller.Detail(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if gotGoodsID != 11 {
		t.Fatalf("goodsID = %d, want 11", gotGoodsID)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["goods_id"] != float64(11) || body["num"] != float64(7) || body["available"] != float64(7) || body["locked"] != float64(2) || body["sold"] != float64(1) {
		t.Fatalf("response body = %+v", body)
	}
}

func TestInventoryControllerReturnsOrderDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotOrderSn string
	controller := NewInventoryController(&fakeInventoryServiceFactory{
		inventory: fakeInventorySrv{
			orderDetail: func(_ context.Context, orderSn string) (*ipb.SellDetailInfo, error) {
				gotOrderSn = orderSn
				return &ipb.SellDetailInfo{
					OrderSn:    "order-1",
					Status:     1,
					StatusName: "reserved",
					GoodsInfo: []*ipb.GoodsInvInfo{
						{GoodsId: 11, Num: 2},
						{GoodsId: 12, Num: 1},
					},
				}, nil
			},
		},
	}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "order_sn", Value: "order-1"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/inventory/orders/order-1", nil)

	controller.OrderDetail(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if gotOrderSn != "order-1" {
		t.Fatalf("orderSn = %s, want order-1", gotOrderSn)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["order_sn"] != "order-1" || body["status"] != float64(1) || body["status_name"] != "reserved" {
		t.Fatalf("response body = %+v", body)
	}
	goods, ok := body["goods"].([]any)
	if !ok || len(goods) != 2 {
		t.Fatalf("goods = %+v", body["goods"])
	}
}

func assertInventoryErrorCode(t *testing.T, body []byte, want int) {
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

type fakeInventoryServiceFactory struct {
	inventory inventorysvc.InventorySrv
}

func (f *fakeInventoryServiceFactory) Goods() goodsv1.GoodsSrv {
	return nil
}

func (f *fakeInventoryServiceFactory) Inventory() inventorysvc.InventorySrv {
	return f.inventory
}

func (f *fakeInventoryServiceFactory) Orders() orderv1.OrderSrv {
	return nil
}

func (f *fakeInventoryServiceFactory) Users() userv1.UserSrv {
	return nil
}

func (f *fakeInventoryServiceFactory) Sms() smsv1.SmsSrv {
	return nil
}

var _ service.ServiceFactory = &fakeInventoryServiceFactory{}

type fakeInventorySrv struct {
	detail      func(context.Context, uint64) (*ipb.GoodsInvInfo, error)
	orderDetail func(context.Context, string) (*ipb.SellDetailInfo, error)
}

func (f fakeInventorySrv) Detail(ctx context.Context, goodsID uint64) (*ipb.GoodsInvInfo, error) {
	if f.detail != nil {
		return f.detail(ctx, goodsID)
	}
	return nil, nil
}

func (f fakeInventorySrv) OrderDetail(ctx context.Context, orderSn string) (*ipb.SellDetailInfo, error) {
	if f.orderDetail != nil {
		return f.orderDetail(ctx, orderSn)
	}
	return nil, nil
}
