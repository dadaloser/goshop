package sms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"goshop/app/goshop/api/internal/captcha"
	"goshop/app/goshop/api/internal/service"
	goodsv1 "goshop/app/goshop/api/internal/service/goods/v1"
	orderv1 "goshop/app/goshop/api/internal/service/order/v1"
	smsv1 "goshop/app/goshop/api/internal/service/sms/v1"
	userv1 "goshop/app/goshop/api/internal/service/user/v1"
	"goshop/app/pkg/code"
	"goshop/gmicro/server/restserver/validation"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func TestSmsControllerSendSmsRejectsMissingServiceFactory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registerMobileValidator(t)

	controller := NewSmsController(nil, nil, &fakeSmsCodeStore{}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = newSendSmsRequest(t)

	controller.SendSms(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertSmsErrorCode(t, recorder.Body.Bytes(), code.ErrConnectGRPC)
}

func TestSmsControllerSendSmsRejectsMissingCodeStoreBeforeSending(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registerMobileValidator(t)

	smsSrv := &fakeSmsSrv{}
	controller := NewSmsController(&fakeSmsServiceFactory{sms: smsSrv}, nil, nil, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = newSendSmsRequest(t)

	controller.SendSms(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	assertSmsErrorCode(t, recorder.Body.Bytes(), code.ErrSmsSend)
	if smsSrv.called {
		t.Fatal("SendSms reached sms service when code store is missing")
	}
}

func TestSmsControllerSendSmsRejectsNilSmsService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registerMobileValidator(t)

	controller := NewSmsController(&fakeSmsServiceFactory{}, nil, &fakeSmsCodeStore{}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = newSendSmsRequest(t)

	controller.SendSms(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertSmsErrorCode(t, recorder.Body.Bytes(), code.ErrConnectGRPC)
}

func TestSmsControllerSendSmsStoresGeneratedCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registerMobileValidator(t)

	store := &fakeSmsCodeStore{}
	smsSrv := &fakeSmsSrv{}
	controller := NewSmsController(&fakeSmsServiceFactory{sms: smsSrv}, nil, store, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = newSendSmsRequest(t)

	controller.SendSms(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !smsSrv.called {
		t.Fatal("SendSms did not call sms service")
	}
	if store.key != "sms:1:13800138000" {
		t.Fatalf("stored key = %q, want sms:1:13800138000", store.key)
	}
	if len(store.value) != 6 {
		t.Fatalf("stored code length = %d, want 6", len(store.value))
	}
	if store.ttl <= 0 {
		t.Fatalf("ttl = %v, want positive", store.ttl)
	}
}

func newSendSmsRequest(t *testing.T) *http.Request {
	t.Helper()

	id, _, answer, err := captcha.NewDigitCaptcha().Generate()
	if err != nil {
		t.Fatalf("generate captcha: %v", err)
	}
	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/sms?mobile=13800138000&type=1&captcha_id="+id+"&captcha="+answer,
		nil,
	)
	return req
}

var registerMobileOnce sync.Once

func registerMobileValidator(t *testing.T) {
	t.Helper()

	var err error
	registerMobileOnce.Do(func() {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			err = v.RegisterValidation("mobile", validation.ValidateMobile)
		}
	})
	if err != nil {
		t.Fatalf("register mobile validator: %v", err)
	}
}

func assertSmsErrorCode(t *testing.T, body []byte, want int) {
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

type fakeSmsServiceFactory struct {
	sms smsv1.SmsSrv
}

func (f *fakeSmsServiceFactory) Goods() goodsv1.GoodsSrv {
	return nil
}

func (f *fakeSmsServiceFactory) Orders() orderv1.OrderSrv {
	return nil
}

func (f *fakeSmsServiceFactory) Users() userv1.UserSrv {
	return nil
}

func (f *fakeSmsServiceFactory) Sms() smsv1.SmsSrv {
	return f.sms
}

var _ service.ServiceFactory = &fakeSmsServiceFactory{}

type fakeSmsSrv struct {
	called bool
	err    error
}

func (f *fakeSmsSrv) SendSms(context.Context, string, string, string) error {
	f.called = true
	return f.err
}

type fakeSmsCodeStore struct {
	key   string
	value string
	ttl   time.Duration
}

func (f *fakeSmsCodeStore) Get(context.Context, string) (string, error) {
	return "", nil
}

func (f *fakeSmsCodeStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	f.key = key
	f.value = value
	f.ttl = ttl
	return nil
}

func (f *fakeSmsCodeStore) Delete(context.Context, string) bool {
	return true
}
