package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goshop/app/pkg/errorcatalog"
	"goshop/app/pkg/options"
	"goshop/pkg/common/core"
	"goshop/pkg/errcode"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	errorcatalog.RegisterAll()
	m.Run()
}

type fakeCallbackService struct {
	calls     int
	duplicate bool
}
type fakeNonceStore struct {
	reserved bool
	calls    int
	err      error
}

func (f *fakeNonceStore) Reserve(context.Context, string, time.Duration) (bool, error) {
	f.calls++
	if f.err != nil {
		return false, f.err
	}
	if f.calls > 1 {
		return false, nil
	}
	return f.reserved, nil
}

func (f *fakeCallbackService) ProcessPayCallback(context.Context, *CallbackRequest) (bool, error) {
	f.calls++
	return f.duplicate, nil
}

func TestCallbackHandlerRequiresValidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Unix(1700000000, 0)
	body := `{"event_id":"evt-1","event_type":"payment_succeeded","order_sn":"order-1","trade_no":"trade-1","amount_fen":100}`
	tests := []struct {
		name, signature string
		want            int
		calls           int
	}{{name: "invalid signature", signature: "00", want: http.StatusUnauthorized}, {name: "valid signature", want: http.StatusOK, calls: 1}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeCallbackService{}
			handler := NewCallbackHandlerWithNonceStore(&options.PaymentOptions{Enabled: true, Provider: "mock", CallbackSecret: "secret", CallbackMaxSkew: time.Minute}, service, &fakeNonceStore{reserved: true})
			handler.now = func() time.Time { return now }
			timestamp := "1700000000"
			signature := tt.signature
			if signature == "" {
				mac := hmac.New(sha256.New, []byte("secret"))
				_, _ = mac.Write([]byte(timestamp + "\nmock\nnonce-1\n"))
				_, _ = mac.Write([]byte(body))
				signature = hex.EncodeToString(mac.Sum(nil))
			}
			router := gin.New()
			router.POST("/callback/:provider", handler.Handle)
			req := httptest.NewRequest(http.MethodPost, "/callback/mock", strings.NewReader(body))
			req.Header.Set("X-Payment-Timestamp", timestamp)
			req.Header.Set("X-Payment-Nonce", "nonce-1")
			req.Header.Set("X-Payment-Signature", signature)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d", rec.Code, tt.want)
			}
			if tt.want != http.StatusOK {
				var response core.ErrResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("decode error response = %v", err)
				}
				if response.Code != errcode.ErrSignatureInvalid {
					t.Fatalf("error code = %d, want %d", response.Code, errcode.ErrSignatureInvalid)
				}
			}
			if service.calls != tt.calls {
				t.Fatalf("calls=%d want=%d", service.calls, tt.calls)
			}
		})
	}
}

func TestCallbackHandlerRejectsExpiredSignature(t *testing.T) {
	handler := NewCallbackHandlerWithNonceStore(&options.PaymentOptions{Enabled: true, CallbackSecret: "secret", CallbackMaxSkew: time.Minute}, &fakeCallbackService{}, &fakeNonceStore{reserved: true})
	handler.now = func() time.Time { return time.Unix(1700001000, 0) }
	if handler.verify("mock", "1700000000", "nonce-1", "00", []byte(`{}`)) {
		t.Fatal("expired callback signature accepted")
	}
}

func TestCallbackHandlerRejectsNonceReplay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Unix(1700000000, 0)
	body := `{"event_id":"evt-1","event_type":"payment_succeeded","order_sn":"order-1","amount_fen":100}`
	timestamp, nonce := "1700000000", "nonce-1"
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte(timestamp + "\nmock\n" + nonce + "\n"))
	_, _ = mac.Write([]byte(body))
	signature := hex.EncodeToString(mac.Sum(nil))
	service := &fakeCallbackService{}
	handler := NewCallbackHandlerWithNonceStore(&options.PaymentOptions{Enabled: true, CallbackSecret: "secret", CallbackMaxSkew: time.Minute}, service, &fakeNonceStore{reserved: true})
	handler.now = func() time.Time { return now }
	router := gin.New()
	router.POST("/callback/:provider", handler.Handle)
	for index, want := range []int{http.StatusOK, http.StatusConflict} {
		req := httptest.NewRequest(http.MethodPost, "/callback/mock", strings.NewReader(body))
		req.Header.Set("X-Payment-Timestamp", timestamp)
		req.Header.Set("X-Payment-Nonce", nonce)
		req.Header.Set("X-Payment-Signature", signature)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("request %d status=%d want=%d", index, rec.Code, want)
		}
	}
	if service.calls != 1 {
		t.Fatalf("service calls=%d want=1", service.calls)
	}
}

func TestCallbackHandlerReturnsDuplicateFlagFromService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Unix(1700000000, 0)
	body := `{"event_id":"evt-1","event_type":"payment_succeeded","order_sn":"order-1","amount_fen":100}`
	timestamp, nonce := "1700000000", "nonce-1"
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte(timestamp + "\nmock\n" + nonce + "\n"))
	_, _ = mac.Write([]byte(body))
	signature := hex.EncodeToString(mac.Sum(nil))

	service := &fakeCallbackService{duplicate: true}
	handler := NewCallbackHandlerWithNonceStore(&options.PaymentOptions{Enabled: true, CallbackSecret: "secret", CallbackMaxSkew: time.Minute}, service, &fakeNonceStore{reserved: true})
	handler.now = func() time.Time { return now }
	router := gin.New()
	router.POST("/callback/:provider", handler.Handle)
	req := httptest.NewRequest(http.MethodPost, "/callback/mock", strings.NewReader(body))
	req.Header.Set("X-Payment-Timestamp", timestamp)
	req.Header.Set("X-Payment-Nonce", nonce)
	req.Header.Set("X-Payment-Signature", signature)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /callback/mock status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"duplicate":true`) {
		t.Fatalf("POST /callback/mock body = %s, want duplicate=true", rec.Body.String())
	}
}

func TestCallbackHandlerHandlesNonceStoreFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Unix(1700000000, 0)
	body := `{"event_id":"evt-1","event_type":"payment_succeeded","order_sn":"order-1","amount_fen":100}`
	timestamp, nonce := "1700000000", "nonce-1"
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte(timestamp + "\nmock\n" + nonce + "\n"))
	_, _ = mac.Write([]byte(body))
	signature := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name   string
		store  NonceStore
		status int
	}{
		{name: "nil store", store: nil, status: http.StatusServiceUnavailable},
		{name: "store error", store: &fakeNonceStore{err: errors.New("redis down")}, status: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeCallbackService{}
			handler := NewCallbackHandlerWithNonceStore(&options.PaymentOptions{Enabled: true, CallbackSecret: "secret", CallbackMaxSkew: time.Minute}, service, tt.store)
			handler.now = func() time.Time { return now }
			router := gin.New()
			router.POST("/callback/:provider", handler.Handle)
			req := httptest.NewRequest(http.MethodPost, "/callback/mock", strings.NewReader(body))
			req.Header.Set("X-Payment-Timestamp", timestamp)
			req.Header.Set("X-Payment-Nonce", nonce)
			req.Header.Set("X-Payment-Signature", signature)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("POST /callback/mock status = %d, want %d, body=%s", rec.Code, tt.status, rec.Body.String())
			}
			if service.calls != 0 {
				t.Fatalf("ProcessPayCallback() calls = %d, want %d", service.calls, 0)
			}
		})
	}
}
