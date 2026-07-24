package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"goshop/app/pkg/options"

	"github.com/google/uuid"
)

type InitiateRequest struct {
	OrderSN   string
	AmountFen int64
	Subject   string
}
type CallbackRequest struct {
	Provider, EventID, EventType, OrderSN, TradeNo string
	AmountFen                                      int64
}
type InitiateResponse struct {
	PaymentID, Provider, CheckoutURL string
	ExpiresAt                        time.Time
}
type RefundRequest struct {
	RequestID string `json:"request_id"`
	OrderSN   string `json:"order_sn"`
	TradeNo   string `json:"trade_no"`
	AmountFen int64  `json:"amount_fen"`
	Reason    string `json:"reason"`
}
type RefundResponse struct {
	ProviderRefundID string `json:"provider_refund_id"`
	Status           string `json:"status"`
}
type Transaction struct {
	EventID    string    `json:"event_id"`
	OrderSN    string    `json:"order_sn"`
	TradeNo    string    `json:"trade_no"`
	EventType  string    `json:"event_type"`
	AmountFen  int64     `json:"amount_fen"`
	OccurredAt time.Time `json:"occurred_at"`
}
type Provider interface {
	Initiate(context.Context, InitiateRequest) (InitiateResponse, error)
	Refund(context.Context, RefundRequest) (RefundResponse, error)
	ListTransactions(context.Context, time.Time, time.Time) ([]Transaction, error)
}
type HMACProvider struct {
	opts   *options.PaymentOptions
	client *http.Client
}

func NewProvider(opts *options.PaymentOptions) Provider {
	timeout := 10 * time.Second
	if opts != nil && opts.RequestTimeout > 0 {
		timeout = opts.RequestTimeout
	}
	return &HMACProvider{opts: opts, client: &http.Client{Timeout: timeout}}
}
func (p *HMACProvider) Initiate(_ context.Context, req InitiateRequest) (InitiateResponse, error) {
	if p == nil || p.opts == nil || req.OrderSN == "" || req.AmountFen <= 0 {
		return InitiateResponse{}, fmt.Errorf("invalid payment initiation")
	}
	id := uuid.NewString()
	expires := time.Now().Add(15 * time.Minute)
	payload := req.OrderSN + "\n" + strconv.FormatInt(req.AmountFen, 10) + "\n" + strconv.FormatInt(expires.Unix(), 10) + "\n" + id
	mac := hmac.New(sha256.New, []byte(p.opts.CallbackSecret))
	_, _ = mac.Write([]byte(payload))
	token := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	checkout, err := url.Parse(p.opts.CheckoutBaseURL)
	if err != nil {
		return InitiateResponse{}, fmt.Errorf("parse checkout URL: %w", err)
	}
	query := checkout.Query()
	query.Set("payment_id", id)
	query.Set("order_sn", req.OrderSN)
	query.Set("amount_fen", strconv.FormatInt(req.AmountFen, 10))
	query.Set("expires_at", strconv.FormatInt(expires.Unix(), 10))
	query.Set("token", token)
	checkout.RawQuery = query.Encode()
	return InitiateResponse{PaymentID: id, Provider: p.opts.Provider, CheckoutURL: checkout.String(), ExpiresAt: expires}, nil
}

func (p *HMACProvider) Refund(ctx context.Context, request RefundRequest) (RefundResponse, error) {
	var response RefundResponse
	if p == nil || p.opts == nil || request.RequestID == "" || request.OrderSN == "" || request.AmountFen <= 0 {
		return response, fmt.Errorf("invalid refund request")
	}
	if err := p.doJSON(ctx, http.MethodPost, p.opts.RefundURL, request, &response); err != nil {
		return response, fmt.Errorf("submit provider refund: %w", err)
	}
	if response.ProviderRefundID == "" || response.Status == "" {
		return RefundResponse{}, fmt.Errorf("provider refund response is incomplete")
	}
	return response, nil
}

func (p *HMACProvider) ListTransactions(ctx context.Context, from, to time.Time) ([]Transaction, error) {
	if p == nil || p.opts == nil || !from.Before(to) {
		return nil, fmt.Errorf("invalid reconciliation window")
	}
	endpoint, err := url.Parse(p.opts.ReconcileURL)
	if err != nil {
		return nil, fmt.Errorf("parse reconciliation URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("from", from.UTC().Format(time.RFC3339))
	query.Set("to", to.UTC().Format(time.RFC3339))
	endpoint.RawQuery = query.Encode()
	var response struct {
		Transactions []Transaction `json:"transactions"`
	}
	if err := p.doJSON(ctx, http.MethodGet, endpoint.String(), nil, &response); err != nil {
		return nil, fmt.Errorf("list provider transactions: %w", err)
	}
	return response.Transactions, nil
}

func (p *HMACProvider) doJSON(ctx context.Context, method, endpoint string, input, output any) error {
	if p == nil || p.opts == nil || p.client == nil || endpoint == "" {
		return fmt.Errorf("payment provider is not configured")
	}
	var body io.Reader
	payload := []byte{}
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		payload = encoded
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(p.opts.CallbackSecret))
	_, _ = mac.Write([]byte(timestamp + "\n" + method + "\n"))
	_, _ = mac.Write(payload)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Payment-Timestamp", timestamp)
	req.Header.Set("X-Payment-Signature", fmt.Sprintf("%x", mac.Sum(nil)))
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("provider returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(output); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
