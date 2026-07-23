package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goshop/app/pkg/options"

	"github.com/gin-gonic/gin"
)

type fakeCallbackService struct {
	calls     int
	duplicate bool
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
			handler := NewCallbackHandler(&options.PaymentOptions{Enabled: true, Provider: "mock", CallbackSecret: "secret", CallbackMaxSkew: time.Minute}, service)
			handler.now = func() time.Time { return now }
			timestamp := "1700000000"
			signature := tt.signature
			if signature == "" {
				mac := hmac.New(sha256.New, []byte("secret"))
				_, _ = mac.Write([]byte(timestamp + "\nmock\n"))
				_, _ = mac.Write([]byte(body))
				signature = hex.EncodeToString(mac.Sum(nil))
			}
			router := gin.New()
			router.POST("/callback/:provider", handler.Handle)
			req := httptest.NewRequest(http.MethodPost, "/callback/mock", strings.NewReader(body))
			req.Header.Set("X-Payment-Timestamp", timestamp)
			req.Header.Set("X-Payment-Signature", signature)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d", rec.Code, tt.want)
			}
			if service.calls != tt.calls {
				t.Fatalf("calls=%d want=%d", service.calls, tt.calls)
			}
		})
	}
}

func TestCallbackHandlerRejectsExpiredSignature(t *testing.T) {
	handler := NewCallbackHandler(&options.PaymentOptions{Enabled: true, CallbackSecret: "secret", CallbackMaxSkew: time.Minute}, &fakeCallbackService{})
	handler.now = func() time.Time { return time.Unix(1700001000, 0) }
	if handler.verify("mock", "1700000000", "00", []byte(`{}`)) {
		t.Fatal("expired callback signature accepted")
	}
}
