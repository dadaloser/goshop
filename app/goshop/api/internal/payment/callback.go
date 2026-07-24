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
		c.JSON(http.StatusServiceUnavailable, gin.H{"msg": "payment callback unavailable"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid callback"})
		return
	}
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	timestamp := c.GetHeader("X-Payment-Timestamp")
	nonce := strings.TrimSpace(c.GetHeader("X-Payment-Nonce"))
	if nonce == "" || !h.verify(provider, timestamp, nonce, c.GetHeader("X-Payment-Signature"), body) {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "invalid callback signature"})
		return
	}
	if h.nonces == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"msg": "callback replay protection unavailable"})
		return
	}
	reserved, reserveErr := h.nonces.Reserve(c, provider+":"+nonce, 2*h.opts.CallbackMaxSkew)
	if reserveErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"msg": "callback replay protection unavailable"})
		return
	}
	if !reserved {
		c.JSON(http.StatusConflict, gin.H{"msg": "callback nonce replayed"})
		return
	}
	var payload callbackPayload
	if err = json.Unmarshal(body, &payload); err != nil || payload.EventID == "" || payload.EventType == "" || payload.OrderSN == "" || payload.AmountFen < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid callback payload"})
		return
	}
	duplicate, err := h.service.ProcessPayCallback(c, &CallbackRequest{Provider: provider, EventID: payload.EventID, EventType: payload.EventType, OrderSN: payload.OrderSN, TradeNo: payload.TradeNo, AmountFen: payload.AmountFen})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"msg": "callback rejected"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "duplicate": duplicate})
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
