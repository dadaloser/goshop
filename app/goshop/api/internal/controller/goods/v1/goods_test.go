package goods

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gpb "goshop/api/goods/v1"
	"goshop/app/goshop/api/internal/service"
	goodsv1 "goshop/app/goshop/api/internal/service/goods/v1"
	inventoryv1 "goshop/app/goshop/api/internal/service/inventory/v1"
	orderv1 "goshop/app/goshop/api/internal/service/order/v1"
	smsv1 "goshop/app/goshop/api/internal/service/sms/v1"
	userv1 "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/pkg/code"

	"github.com/gin-gonic/gin"
)

func TestGoodsControllerListRejectsMissingService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := NewGoodsController(nil, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/goods", nil)

	controller.List(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertErrorCode(t, recorder.Body.Bytes(), code.ErrConnectGRPC)
}

func TestGoodsControllerListRejectsNilGoodsService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := NewGoodsController(&fakeGoodsServiceFactory{}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/goods", nil)

	controller.List(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertErrorCode(t, recorder.Body.Bytes(), code.ErrConnectGRPC)
}

func TestGoodsControllerListRejectsNilGoodsResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := NewGoodsController(&fakeGoodsServiceFactory{goods: &fakeGoodsSrv{}}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/goods", nil)

	controller.List(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertErrorCode(t, recorder.Body.Bytes(), code.ErrConnectGRPC)
}

func TestGoodsControllerListHandlesPartialGoodsData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	goodsSrv := &fakeGoodsSrv{
		resp: &gpb.GoodsListResponse{
			Total: 2,
			Data: []*gpb.GoodsInfoResponse{
				nil,
				{
					Id:        11,
					Name:      "keyboard",
					ShopPrice: 99,
				},
			},
		},
	}
	controller := NewGoodsController(&fakeGoodsServiceFactory{goods: goodsSrv}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/goods?p=3&pnum=12", nil)

	controller.List(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if goodsSrv.gotRequest.GetPages() != 3 || goodsSrv.gotRequest.GetPagePerNums() != 12 {
		t.Fatalf("request page = (%d,%d), want (3,12)", goodsSrv.gotRequest.GetPages(), goodsSrv.gotRequest.GetPagePerNums())
	}

	var body struct {
		Total int                      `json:"total"`
		Data  []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Total != 2 {
		t.Fatalf("total = %d, want 2", body.Total)
	}
	if len(body.Data) != 1 {
		t.Fatalf("data length = %d, want 1", len(body.Data))
	}
	if body.Data[0]["name"] != "keyboard" {
		t.Fatalf("name = %v, want keyboard", body.Data[0]["name"])
	}
}

func assertErrorCode(t *testing.T, body []byte, want int) {
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

type fakeGoodsServiceFactory struct {
	goods goodsv1.GoodsSrv
}

func (f *fakeGoodsServiceFactory) Goods() goodsv1.GoodsSrv {
	return f.goods
}

func (f *fakeGoodsServiceFactory) Orders() orderv1.OrderSrv {
	return nil
}

func (f *fakeGoodsServiceFactory) Inventory() inventoryv1.InventorySrv {
	return nil
}

func (f *fakeGoodsServiceFactory) Users() userv1.UserSrv {
	return nil
}

func (f *fakeGoodsServiceFactory) Sms() smsv1.SmsSrv {
	return nil
}

var _ service.ServiceFactory = &fakeGoodsServiceFactory{}

type fakeGoodsSrv struct {
	resp       *gpb.GoodsListResponse
	err        error
	gotRequest *gpb.GoodsFilterRequest
}

func (f *fakeGoodsSrv) List(ctx context.Context, request *gpb.GoodsFilterRequest) (*gpb.GoodsListResponse, error) {
	f.gotRequest = request
	return f.resp, f.err
}
