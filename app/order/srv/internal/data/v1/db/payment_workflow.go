package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goshop/app/order/srv/internal/domain/do"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (o *orders) ClaimRefundJobs(ctx context.Context, limit, maxAttempts int, lockTimeout time.Duration) ([]do.RefundJob, error) {
	if limit <= 0 {
		limit = 20
	}
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	if lockTimeout <= 0 {
		lockTimeout = 2 * time.Minute
	}
	now := time.Now().UTC()
	jobs := make([]do.RefundJob, 0, limit)
	err := o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var outboxes []do.RefundOutboxDO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("(attempts < ? AND available_at <= ? AND status IN ?) OR (status = ? AND locked_at < ?)", maxAttempts, now, []string{"pending", "retry"}, "processing", now.Add(-lockTimeout)).
			Order("id ASC").Limit(limit).Find(&outboxes).Error; err != nil {
			return err
		}
		for i := range outboxes {
			outbox := &outboxes[i]
			if err := tx.Model(outbox).Updates(map[string]interface{}{"status": "processing", "attempts": gorm.Expr("attempts + 1"), "locked_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
			var row struct {
				RefundRequestID                         uint64
				OrderSN, TradeNo, Reason, CorrelationID string
				AmountFen                               int64
			}
			result := tx.Table("order_refund_requests AS r").Select("r.id AS refund_request_id, r.order_sn, o.trade_no, r.amount_fen, r.reason, r.correlation_id").Joins("JOIN orderinfo AS o ON o.order_sn = r.order_sn").Where("r.id = ?", outbox.RefundRequestID).Scan(&row)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("refund request %d has no order", outbox.RefundRequestID)
			}
			jobs = append(jobs, do.RefundJob{OutboxID: outbox.ID, RefundRequestID: row.RefundRequestID, OrderSN: row.OrderSN, TradeNo: row.TradeNo, AmountFen: row.AmountFen, Reason: row.Reason, CorrelationID: row.CorrelationID, Attempts: outbox.Attempts + 1})
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return jobs, nil
}

func (o *orders) CompleteRefundJob(ctx context.Context, id uint64, success bool, provider, providerRefundID, providerStatus, detail string, maxAttempts int) error {
	if id == 0 {
		return errors.WithCode(code2.ErrValidation, "refund job id is required")
	}
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	now := time.Now().UTC()
	err := o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var outbox do.RefundOutboxDO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", id, "processing").First(&outbox).Error; err != nil {
			return err
		}
		if success {
			status := "PROCESSING"
			orderStatus := OrderStatusRefundPendingDB
			if strings.EqualFold(providerStatus, "refunded") || strings.EqualFold(providerStatus, "succeeded") {
				status = "REFUNDED"
				orderStatus = OrderStatusRefundedDB
			}
			if err := tx.Model(&do.RefundRequestDO{}).Where("id = ?", outbox.RefundRequestID).Updates(map[string]interface{}{"status": status, "provider": provider, "provider_refund_id": providerRefundID, "failure_reason": "", "updated_at": now}).Error; err != nil {
				return err
			}
			var refund do.RefundRequestDO
			if err := tx.Where("id = ?", outbox.RefundRequestID).First(&refund).Error; err != nil {
				return err
			}
			if orderStatus == OrderStatusRefundedDB {
				if err := updateRefundOrderStatus(tx, refund.OrderSN, OrderStatusRefundPendingDB, orderStatus, "provider refund completed"); err != nil {
					return err
				}
			}
			return tx.Model(&outbox).Updates(map[string]interface{}{"status": "completed", "locked_at": nil, "last_error": "", "updated_at": now}).Error
		}
		detail = truncatePaymentDetail(detail)
		decision := refundFailureDecision(outbox.Attempts, maxAttempts, now)
		if decision.dead {
			if err := tx.Model(&do.RefundRequestDO{}).Where("id = ?", outbox.RefundRequestID).Updates(map[string]interface{}{"status": "FAILED", "failure_reason": detail, "updated_at": now}).Error; err != nil {
				return err
			}
			var refund do.RefundRequestDO
			if err := tx.Where("id = ?", outbox.RefundRequestID).First(&refund).Error; err != nil {
				return err
			}
			if err := updateRefundOrderStatus(tx, refund.OrderSN, OrderStatusRefundPendingDB, OrderStatusRefundFailedDB, "refund retry limit reached"); err != nil {
				return err
			}
			return tx.Model(&outbox).Updates(map[string]interface{}{"status": "dead", "locked_at": nil, "last_error": detail, "updated_at": now}).Error
		}
		return tx.Model(&outbox).Updates(map[string]interface{}{"status": "retry", "available_at": decision.availableAt, "locked_at": nil, "last_error": detail, "updated_at": now}).Error
	})
	if err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

type refundFailureResult struct {
	dead        bool
	availableAt time.Time
}

func refundFailureDecision(attempts, maxAttempts int, now time.Time) refundFailureResult {
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	if attempts >= maxAttempts {
		return refundFailureResult{dead: true}
	}
	backoff := time.Duration(1<<min(attempts, 10)) * time.Second
	return refundFailureResult{availableAt: now.Add(backoff)}
}

func updateRefundOrderStatus(tx *gorm.DB, orderSN, from, to, reason string) error {
	var order do.OrderInfoDO
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_sn = ?", orderSN).First(&order).Error; err != nil {
		return err
	}
	if order.Status == to {
		return nil
	}
	if order.Status != from {
		return fmt.Errorf("order %s status is %s, expected %s", orderSN, order.Status, from)
	}
	if err := tx.Model(&order).Update("status", to).Error; err != nil {
		return err
	}
	return tx.Create(&do.OrderStatusLogDO{OrderID: order.ID, OrderSn: order.OrderSn, FromStatus: from, ToStatus: to, Reason: reason, Source: "payment.refund.worker", Operator: "system"}).Error
}

const (
	OrderStatusRefundPendingDB = "REFUND_PENDING"
	OrderStatusRefundedDB      = "REFUNDED"
	OrderStatusRefundFailedDB  = "REFUND_FAILED"
)

func truncatePaymentDetail(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 255 {
		return value[:255]
	}
	return value
}

func (o *orders) ReconcilePayments(ctx context.Context, provider string, from, to time.Time, transactions []do.PaymentEventDO) (*do.PaymentReconciliationRunDO, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" || !from.Before(to) {
		return nil, errors.WithCode(code2.ErrValidation, "invalid reconciliation request")
	}
	if len(transactions) > 10000 {
		return nil, errors.WithCode(code2.ErrValidation, "reconciliation batch is too large")
	}
	for _, transaction := range transactions {
		if strings.TrimSpace(transaction.EventID) == "" || strings.TrimSpace(transaction.OrderSN) == "" || strings.TrimSpace(transaction.EventType) == "" || transaction.ProviderAmountFen < 0 {
			return nil, errors.WithCode(code2.ErrValidation, "provider transaction is invalid")
		}
	}
	providerEvents := make(map[string]struct{}, len(transactions))
	for _, transaction := range transactions {
		if _, exists := providerEvents[transaction.EventID]; exists {
			return nil, errors.WithCode(code2.ErrValidation, "provider statement contains duplicate event_id")
		}
		providerEvents[transaction.EventID] = struct{}{}
	}
	now := time.Now().UTC()
	run := &do.PaymentReconciliationRunDO{Provider: provider, WindowStart: from.UTC(), WindowEnd: to.UTC(), StartedAt: now, Status: "processing"}
	err := o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(run).Error; err != nil {
			return err
		}
		var locals []do.PaymentEventDO
		if err := tx.Where("provider = ? AND received_at >= ? AND received_at < ?", provider, from.UTC(), to.UTC()).Find(&locals).Error; err != nil {
			return err
		}
		localByEvent := make(map[string]do.PaymentEventDO, len(locals))
		for _, local := range locals {
			localByEvent[local.EventID] = local
		}
		seen := make(map[string]struct{}, len(transactions))
		items := make([]do.PaymentReconciliationItemDO, 0, len(transactions)+len(locals))
		mismatches := 0
		for _, remote := range transactions {
			seen[remote.EventID] = struct{}{}
			result, detail, localAmount := "matched", "", int64(0)
			local, ok := localByEvent[remote.EventID]
			if !ok {
				result, detail = "missing_local", "provider transaction missing locally"
			} else {
				localAmount = local.ProviderAmountFen
				if local.OrderSN != remote.OrderSN || local.TradeNo != remote.TradeNo || local.EventType != remote.EventType || local.ProviderAmountFen != remote.ProviderAmountFen {
					result, detail = "mismatch", "provider and local transaction differ"
				}
			}
			if result != "matched" {
				mismatches++
			}
			items = append(items, do.PaymentReconciliationItemDO{RunID: run.ID, ProviderEventID: remote.EventID, OrderSN: remote.OrderSN, TradeNo: remote.TradeNo, EventType: remote.EventType, ProviderAmountFen: remote.ProviderAmountFen, LocalAmountFen: localAmount, Result: result, Detail: detail, CreatedAt: now})
		}
		for _, local := range locals {
			if _, ok := seen[local.EventID]; ok {
				continue
			}
			mismatches++
			items = append(items, do.PaymentReconciliationItemDO{RunID: run.ID, ProviderEventID: local.EventID, OrderSN: local.OrderSN, TradeNo: local.TradeNo, EventType: local.EventType, LocalAmountFen: local.ProviderAmountFen, Result: "missing_provider", Detail: "local transaction missing from provider statement", CreatedAt: now})
		}
		if len(items) > 0 {
			if err := tx.CreateInBatches(items, 100).Error; err != nil {
				return err
			}
		}
		finished := time.Now().UTC()
		run.CheckedCount = len(items)
		run.MismatchCount = mismatches
		run.Status = "completed"
		run.FinishedAt = &finished
		return tx.Model(run).Updates(map[string]interface{}{"checked_count": run.CheckedCount, "mismatch_count": mismatches, "status": run.Status, "finished_at": finished}).Error
	})
	if err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, fmt.Sprintf("reconcile payments: %v", err))
	}
	return run, nil
}
