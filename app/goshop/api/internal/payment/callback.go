package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goshop/app/pkg/options"
	"goshop/pkg/common/core"
	"goshop/pkg/errcode"
	apperrors "goshop/pkg/errors"

	"github.com/gin-gonic/gin"
)

type CallbackService interface {
	ProcessPayCallback(ctx context.Context, req *CallbackRequest) (bool, error)
}
type CallbackHandler struct {
	opts    *options.PaymentOptions
	service CallbackService
	nonces  NonceStore
	now     func() time.Time
}

func NewCallbackHandler(opts *options.PaymentOptions, service CallbackService) *CallbackHandler {
	return NewCallbackHandlerWithNonceStore(opts, service, NewRedisNonceStore())
}

func NewCallbackHandlerWithNonceStore(opts *options.PaymentOptions, service CallbackService, nonces NonceStore) *CallbackHandler {
	return &CallbackHandler{opts: opts, service: service, nonces: nonces, now: time.Now}
}

type callbackPayload struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	OrderSN   string `json:"order_sn"`
	TradeNo   string `json:"trade_no"`
	AmountFen int64  `json:"amount_fen"`
}

func (h *CallbackHandler) Handle(c *gin.Context) {
	if h == nil || h.opts == nil || !h.opts.Enabled || h.service == nil {
		metricPaymentCallbackHTTP.Inc("unavailable")
		core.WriteError(c, apperrors.NewCode(errcode.ErrServiceUnavailable, "payment callback unavailable"))
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		metricPaymentCallbackHTTP.Inc("invalid_body")
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid callback body"))
		return
	}
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	timestamp := c.GetHeader("X-Payment-Timestamp")
	nonce := strings.TrimSpace(c.GetHeader("X-Payment-Nonce"))
	if nonce == "" || !h.verify(provider, timestamp, nonce, c.GetHeader("X-Payment-Signature"), body) {
		metricPaymentCallbackHTTP.Inc("invalid_signature")
		core.WriteError(c, apperrors.NewCode(errcode.ErrSignatureInvalid, "invalid callback signature"))
		return
	}
	if h.nonces == nil {
		metricPaymentCallbackHTTP.Inc("nonce_store_unavailable")
		core.WriteError(c, apperrors.NewCode(errcode.ErrServiceUnavailable, "callback replay protection unavailable"))
		return
	}
	reserved, reserveErr := h.nonces.Reserve(c, provider+":"+nonce, 2*h.opts.CallbackMaxSkew)
	if reserveErr != nil {
		metricPaymentCallbackHTTP.Inc("nonce_store_error")
		core.WriteError(c, apperrors.NewCode(errcode.ErrServiceUnavailable, "callback replay protection unavailable"))
		return
	}
	if !reserved {
		metricPaymentCallbackHTTP.Inc("duplicate")
		core.WriteError(c, apperrors.NewCode(errcode.ErrConflict, "callback nonce replayed"))
		return
	}
	var payload callbackPayload
	if err = json.Unmarshal(body, &payload); err != nil || payload.EventID == "" || payload.EventType == "" || payload.OrderSN == "" || payload.AmountFen < 0 {
		metricPaymentCallbackHTTP.Inc("invalid_payload")
		core.WriteError(c, apperrors.NewCode(errcode.ErrValidation, "invalid callback payload"))
		return
	}
	duplicate, err := h.service.ProcessPayCallback(c, &CallbackRequest{Provider: provider, EventID: payload.EventID, EventType: payload.EventType, OrderSN: payload.OrderSN, TradeNo: payload.TradeNo, AmountFen: payload.AmountFen})
	if err != nil {
		metricPaymentCallbackHTTP.Inc("rejected")
		core.WriteError(c, apperrors.WrapCode(err, errcode.ErrConflict, "payment callback rejected"))
		return
	}
	if duplicate {
		metricPaymentCallbackHTTP.Inc("duplicate")
	} else {
		metricPaymentCallbackHTTP.Inc("accepted")
	}
	c.JSON(http.StatusOK, gin.H{"msg": true, "duplicate": duplicate})
}
func (h *CallbackHandler) verify(provider, timestamp, nonce, signature string, body []byte) bool {
	unix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	signedAt := time.Unix(unix, 0)
	if delta := h.now().Sub(signedAt); delta > h.opts.CallbackMaxSkew || delta < -h.opts.CallbackMaxSkew {
		return false
	}
	provided, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(h.opts.CallbackSecret))
	_, _ = mac.Write([]byte(timestamp + "\n" + provider + "\n" + nonce + "\n"))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}
