package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
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
type Provider interface {
	Initiate(context.Context, InitiateRequest) (InitiateResponse, error)
}
type HMACProvider struct{ opts *options.PaymentOptions }

func NewProvider(opts *options.PaymentOptions) Provider { return &HMACProvider{opts: opts} }
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
