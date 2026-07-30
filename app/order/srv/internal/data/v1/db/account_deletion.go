package db

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"goshop/app/order/srv/internal/domain/do"
	"goshop/app/pkg/accountdeletion"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReviewAccountDeletion persists exactly one decision for a deletion request.
// It is safe to call repeatedly after at-least-once message delivery.
func ReviewAccountDeletion(ctx context.Context, db *gorm.DB, request accountdeletion.Requested) (*accountdeletion.Decision, error) {
	if db == nil || strings.TrimSpace(request.EventID) == "" || request.UserID == 0 {
		return nil, gorm.ErrInvalidData
	}
	decision := &accountdeletion.Decision{}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inbox do.AccountDeletionInboxDO
		err := tx.Where("request_id = ?", request.EventID).First(&inbox).Error
		if err == nil {
			decision = &accountdeletion.Decision{RequestID: inbox.RequestID, UserID: uint64(inbox.UserID), Confirmed: inbox.Decision == "CONFIRMED", Reason: inbox.Reason, DecidedAt: inbox.CreatedAt}
			return nil
		}
		if !strings.Contains(err.Error(), "record not found") {
			return err
		}

		var orders []string
		if err = tx.Model(&do.OrderInfoDO{}).Where("user = ? AND deleted_at IS NULL", request.UserID).Pluck("status", &orders).Error; err != nil {
			return err
		}
		var refunds []string
		if err = tx.Model(&do.RefundRequestDO{}).Joins("JOIN orderinfo ON orderinfo.order_sn = order_refund_requests.order_sn").Where("orderinfo.user = ?", request.UserID).Pluck("order_refund_requests.status", &refunds).Error; err != nil {
			return err
		}
		confirmed, reason := accountDeletionCanProceed(orders, refunds)
		value := "REJECTED"
		subject := accountdeletion.SubjectRejected
		if confirmed {
			value = "CONFIRMED"
			subject = accountdeletion.SubjectConfirmed
		}
		now := time.Now().UTC()
		inbox = do.AccountDeletionInboxDO{RequestID: request.EventID, UserID: int32(request.UserID), Decision: value, Reason: reason, CreatedAt: now}
		if err = tx.Create(&inbox).Error; err != nil {
			return err
		}
		decision = &accountdeletion.Decision{EventID: uuid.NewString(), RequestID: request.EventID, UserID: request.UserID, Confirmed: confirmed, Reason: reason, DecidedAt: now}
		payload, err := json.Marshal(decision)
		if err != nil {
			return err
		}
		return tx.Create(&do.AccountDeletionOutboxDO{ID: decision.EventID, RequestID: request.EventID, EventType: subject, UserID: int32(request.UserID), Payload: payload, Status: "PENDING", AvailableAt: now, CreatedAt: now, UpdatedAt: now}).Error
	})
	return decision, err
}

func accountDeletionCanProceed(orders, refunds []string) (bool, string) {
	for _, status := range orders {
		switch strings.ToUpper(strings.TrimSpace(status)) {
		case "TRADE_CLOSED", "TRADE_FINISHED", "REFUNDED", "REFUND_FAILED":
		default:
			return false, "存在未完成订单或退款，暂不能注销"
		}
	}
	for _, status := range refunds {
		switch strings.ToUpper(strings.TrimSpace(status)) {
		case "REFUNDED", "FAILED", "CLOSED", "CANCELLED":
		default:
			return false, "存在未完成订单或退款，暂不能注销"
		}
	}
	return true, ""
}
