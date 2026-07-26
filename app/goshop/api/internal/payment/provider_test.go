package payment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"goshop/app/pkg/options"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newProviderForTest(opts *options.PaymentOptions, transport roundTripFunc) *HMACProvider {
	provider := NewProvider(opts)
	hmacProvider, ok := provider.(*HMACProvider)
	if !ok {
		panic("NewProvider() returned unexpected provider type")
	}
	hmacProvider.client = &http.Client{
		Timeout:   opts.RequestTimeout,
		Transport: transport,
	}
	return hmacProvider
}

func jsonResponse(status int, body any) *http.Response {
	payload, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(payload))),
	}
}

func TestHMACProviderRefundAndReconciliation(t *testing.T) {
	provider := newProviderForTest(
		&options.PaymentOptions{
			CallbackSecret: "secret",
			RefundURL:      "https://provider.example.test/refunds",
			ReconcileURL:   "https://provider.example.test/transactions",
			RequestTimeout: time.Second,
		},
		func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Payment-Signature") == "" {
				t.Fatal("provider request is unsigned")
			}
			switch r.URL.Path {
			case "/refunds":
				var request RefundRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatalf("json.NewDecoder(...).Decode() error = %v", err)
				}
				if request.RequestID != "refund-1" || request.AmountFen != 100 {
					t.Fatalf("Refund() request = %+v, want request_id=%q amount_fen=%d", request, "refund-1", 100)
				}
				return jsonResponse(http.StatusOK, RefundResponse{ProviderRefundID: "provider-refund-1", Status: "accepted"}), nil
			case "/transactions":
				return jsonResponse(http.StatusOK, map[string]any{"transactions": []Transaction{{EventID: "event-1", OrderSN: "order-1", AmountFen: 100, OccurredAt: time.Unix(100, 0)}}}), nil
			default:
				t.Fatalf("RoundTrip() path = %q, want %q or %q", r.URL.Path, "/refunds", "/transactions")
				return nil, nil
			}
		},
	)
	refund, err := provider.Refund(context.Background(), RefundRequest{RequestID: "refund-1", OrderSN: "order-1", AmountFen: 100})
	if err != nil {
		t.Fatalf("Refund() error = %v", err)
	}
	if refund.ProviderRefundID != "provider-refund-1" {
		t.Fatalf("Refund().ProviderRefundID = %q, want %q", refund.ProviderRefundID, "provider-refund-1")
	}
	transactions, err := provider.ListTransactions(context.Background(), time.Unix(0, 0), time.Unix(200, 0))
	if err != nil {
		t.Fatalf("ListTransactions() error = %v", err)
	}
	if len(transactions) != 1 || transactions[0].EventID != "event-1" {
		t.Fatalf("ListTransactions() = %+v, want one event with id %q", transactions, "event-1")
	}
}

func TestHMACProviderRefundRejectsHTTPFailure(t *testing.T) {
	provider := newProviderForTest(
		&options.PaymentOptions{CallbackSecret: "secret", RefundURL: "https://provider.example.test/refunds", RequestTimeout: time.Second},
		func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"detail":"secret provider detail"}`)),
			}, nil
		},
	)
	if _, err := provider.Refund(context.Background(), RefundRequest{RequestID: "refund-1", OrderSN: "order-1", AmountFen: 100}); err == nil {
		t.Fatal("Refund() error=nil")
	}
}

func TestHMACProviderRefundHonorsTimeout(t *testing.T) {
	provider := newProviderForTest(
		&options.PaymentOptions{CallbackSecret: "secret", RefundURL: "https://provider.example.test/refunds", RequestTimeout: 5 * time.Millisecond},
		func(req *http.Request) (*http.Response, error) {
			<-req.Context().Done()
			return nil, req.Context().Err()
		},
	)
	if _, err := provider.Refund(context.Background(), RefundRequest{RequestID: "refund-timeout", OrderSN: "order-1", AmountFen: 100}); err == nil {
		t.Fatal("Refund() timeout error=nil")
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Refund() error = %v, want deadline exceeded", err)
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
