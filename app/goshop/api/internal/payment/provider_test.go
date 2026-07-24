package payment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goshop/app/pkg/options"
)

func TestHMACProviderRefundAndReconciliation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Payment-Signature") == "" {
			t.Error("provider request is unsigned")
		}
		switch r.URL.Path {
		case "/refunds":
			var request RefundRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
			}
			if request.RequestID != "refund-1" || request.AmountFen != 100 {
				t.Errorf("refund request=%+v", request)
			}
			_ = json.NewEncoder(w).Encode(RefundResponse{ProviderRefundID: "provider-refund-1", Status: "accepted"})
		case "/transactions":
			_ = json.NewEncoder(w).Encode(map[string]any{"transactions": []Transaction{{EventID: "event-1", OrderSN: "order-1", AmountFen: 100, OccurredAt: time.Unix(100, 0)}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	provider := NewProvider(&options.PaymentOptions{CallbackSecret: "secret", RefundURL: server.URL + "/refunds", ReconcileURL: server.URL + "/transactions", RequestTimeout: time.Second})
	refund, err := provider.Refund(context.Background(), RefundRequest{RequestID: "refund-1", OrderSN: "order-1", AmountFen: 100})
	if err != nil {
		t.Fatal(err)
	}
	if refund.ProviderRefundID != "provider-refund-1" {
		t.Fatalf("provider refund id=%q", refund.ProviderRefundID)
	}
	transactions, err := provider.ListTransactions(context.Background(), time.Unix(0, 0), time.Unix(200, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(transactions) != 1 || transactions[0].EventID != "event-1" {
		t.Fatalf("transactions=%+v", transactions)
	}
}

func TestHMACProviderRefundRejectsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "secret provider detail", http.StatusBadGateway)
	}))
	defer server.Close()
	provider := NewProvider(&options.PaymentOptions{CallbackSecret: "secret", RefundURL: server.URL, RequestTimeout: time.Second})
	if _, err := provider.Refund(context.Background(), RefundRequest{RequestID: "refund-1", OrderSN: "order-1", AmountFen: 100}); err == nil {
		t.Fatal("Refund() error=nil")
	}
}

func TestHMACProviderRefundHonorsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(RefundResponse{ProviderRefundID: "late", Status: "accepted"})
	}))
	defer server.Close()
	provider := NewProvider(&options.PaymentOptions{CallbackSecret: "secret", RefundURL: server.URL, RequestTimeout: 5 * time.Millisecond})
	if _, err := provider.Refund(context.Background(), RefundRequest{RequestID: "refund-timeout", OrderSN: "order-1", AmountFen: 100}); err == nil {
		t.Fatal("Refund() timeout error=nil")
	}
}

func TestHMACProviderInitiateAndValidateInputs(t *testing.T) {
	provider := NewProvider(&options.PaymentOptions{Provider: "mock", CheckoutBaseURL: "https://payments.example.test/checkout", CallbackSecret: "secret"})
	result, err := provider.Initiate(context.Background(), InitiateRequest{OrderSN: "order-1", AmountFen: 100, Subject: "goshop order"})
	if err != nil {
		t.Fatalf("Initiate() error = %v", err)
	}
	if result.PaymentID == "" || result.Provider != "mock" || !strings.Contains(result.CheckoutURL, "order_sn=order-1") {
		t.Fatalf("Initiate() response = %+v", result)
	}
	if _, err := provider.Initiate(context.Background(), InitiateRequest{}); err == nil {
		t.Fatal("Initiate() invalid input error=nil")
	}
}

func TestHMACProviderRejectsInvalidConfiguration(t *testing.T) {
	provider := &HMACProvider{}
	if _, err := provider.ListTransactions(context.Background(), time.Unix(1, 0), time.Unix(0, 0)); err == nil {
		t.Fatal("ListTransactions() invalid window error=nil")
	}
	if err := provider.doJSON(context.Background(), http.MethodGet, "", nil, &struct{}{}); err == nil {
		t.Fatal("doJSON() invalid configuration error=nil")
	}
}
