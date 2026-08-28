package db

import (
	"context"
	"database/sql"
	"fmt"
	"goshop/app/pkg/bizcode"
	"strings"
	"time"

	"goshop/app/order/srv/internal/domain/do"
	"goshop/pkg/errcode"
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
		return nil, wrapDatabaseError(err, "database operation")
	}
	return jobs, nil
}

func (o *orders) CompleteRefundJob(ctx context.Context, id uint64, success bool, provider, providerRefundID, providerStatus, detail string, maxAttempts int) error {
	if id == 0 {
		return errors.NewCode(errcode.ErrValidation, "refund job id is required")
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
		return wrapDatabaseError(err, "database operation")
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
		return nil, errors.NewCode(errcode.ErrValidation, "invalid reconciliation request")
	}
	if len(transactions) > 10000 {
		return nil, errors.NewCode(errcode.ErrValidation, "reconciliation batch is too large")
	}
	for _, transaction := range transactions {
		if strings.TrimSpace(transaction.EventID) == "" || strings.TrimSpace(transaction.OrderSN) == "" || strings.TrimSpace(transaction.EventType) == "" || transaction.ProviderAmountFen < 0 {
			return nil, errors.NewCode(errcode.ErrValidation, "provider transaction is invalid")
		}
	}
	providerEvents := make(map[string]struct{}, len(transactions))
	for _, transaction := range transactions {
		if _, exists := providerEvents[transaction.EventID]; exists {
			return nil, errors.NewCode(errcode.ErrValidation, "provider statement contains duplicate event_id")
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
		return nil, wrapDatabaseError(err, "reconcile payments")
	}
	return run, nil
}

func (o *orders) ListPaymentReconciliationRuns(ctx context.Context, provider string, from, to *time.Time, offset, limit int) ([]do.PaymentReconciliationRunDO, int64, error) {
	query := o.db.WithContext(ctx).Model(&do.PaymentReconciliationRunDO{})
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if from != nil {
		query = query.Where("window_start >= ?", from.UTC())
	}
	if to != nil {
		query = query.Where("window_end <= ?", to.UTC())
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, wrapDatabaseError(err, "database operation")
	}
	items := make([]do.PaymentReconciliationRunDO, 0, limit)
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, wrapDatabaseError(err, "database operation")
	}
	return items, total, nil
}

func (o *orders) ListPaymentReconciliationItems(ctx context.Context, provider string, from, to *time.Time, result string, runID uint64, offset, limit int) ([]do.PaymentReconciliationItemDO, int64, error) {
	query := o.db.WithContext(ctx).Table("payment_reconciliation_items AS i").Joins("JOIN payment_reconciliation_runs AS r ON r.id = i.run_id")
	if provider != "" {
		query = query.Where("r.provider = ?", provider)
	}
	if from != nil {
		query = query.Where("r.window_start >= ?", from.UTC())
	}
	if to != nil {
		query = query.Where("r.window_end <= ?", to.UTC())
	}
	if result != "" {
		query = query.Where("i.result = ?", result)
	}
	if runID > 0 {
		query = query.Where("i.run_id = ?", runID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, wrapDatabaseError(err, "database operation")
	}
	items := make([]do.PaymentReconciliationItemDO, 0, limit)
	if err := query.Select("i.*").Order("i.id DESC").Offset(offset).Limit(limit).Scan(&items).Error; err != nil {
		return nil, 0, wrapDatabaseError(err, "database operation")
	}
	return items, total, nil
}

func (o *orders) RetryDeadRefundJob(ctx context.Context, id uint64) (*do.RefundJob, error) {
	if id == 0 {
		return nil, errors.NewCode(errcode.ErrValidation, "refund job id is required")
	}
	now := time.Now().UTC()
	job := &do.RefundJob{}
	err := o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var outbox do.RefundOutboxDO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", id, "dead").First(&outbox).Error; err != nil {
			return err
		}
		var refund do.RefundRequestDO
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", outbox.RefundRequestID).First(&refund).Error; err != nil {
			return err
		}
		if refund.Status == "FAILED" {
			if err := tx.Model(&refund).Updates(map[string]interface{}{"status": "PROCESSING", "failure_reason": "", "updated_at": now}).Error; err != nil {
				return err
			}
		}
		var order do.OrderInfoDO
		if err := tx.Where("order_sn = ?", refund.OrderSN).First(&order).Error; err != nil {
			return err
		}
		if order.Status == OrderStatusRefundFailedDB {
			if err := updateRefundOrderStatus(tx, refund.OrderSN, OrderStatusRefundFailedDB, OrderStatusRefundPendingDB, "refund dead-letter retry requested"); err != nil {
				return err
			}
		}
		if err := tx.Model(&outbox).Updates(map[string]interface{}{"status": "retry", "attempts": 0, "available_at": now, "locked_at": nil, "last_error": "", "updated_at": now}).Error; err != nil {
			return err
		}
		job.OutboxID = outbox.ID
		job.RefundRequestID = refund.ID
		job.OrderSN = refund.OrderSN
		job.AmountFen = refund.AmountFen
		job.Reason = refund.Reason
		job.CorrelationID = refund.CorrelationID
		job.Attempts = 0
		job.TradeNo = order.TradeNo
		return nil
	})
	if err != nil {
		return nil, wrapDatabaseError(err, "database operation")
	}
	return job, nil
}

func (o *orders) GetOrderTrace(ctx context.Context, lookup do.OrderTraceLookup) (*do.OrderTrace, error) {
	order, matchedBy, matchedValue, err := o.resolveTraceOrder(ctx, lookup)
	if err != nil {
		return nil, err
	}

	statusLogs, err := o.loadTraceStatusLogs(ctx, order.OrderSn)
	if err != nil {
		return nil, err
	}
	paymentEvents, _, _, err := o.ListPaymentEvents(ctx, order.OrderSn, 0, 100, false)
	if err != nil {
		return nil, err
	}
	refunds, err := o.loadTraceRefunds(ctx, order.OrderSn)
	if err != nil {
		return nil, err
	}

	return &do.OrderTrace{
		MatchedBy:     matchedBy,
		MatchedValue:  matchedValue,
		Order:         order,
		StatusLogs:    statusLogs,
		PaymentEvents: paymentEvents,
		Refunds:       refunds,
	}, nil
}

func (o *orders) resolveTraceOrder(ctx context.Context, lookup do.OrderTraceLookup) (*do.OrderInfoDO, string, string, error) {
	if lookup.OrderSN != "" {
		order, err := o.Get(ctx, lookup.OrderSN)
		if err != nil {
			return nil, "", "", err
		}
		return order, "order_sn", lookup.OrderSN, nil
	}

	var orderSN string
	query := o.db.WithContext(ctx)
	switch {
	case lookup.TradeNo != "":
		if err := query.Table("orderinfo").Select("order_sn").Where("trade_no = ?", lookup.TradeNo).Limit(1).Scan(&orderSN).Error; err != nil {
			return nil, "", "", wrapDatabaseError(err, "database operation")
		}
		if strings.TrimSpace(orderSN) == "" {
			return nil, "", "", errors.NewCode(bizcode.ErrOrderNotFound, "order not found")
		}
		order, err := o.Get(ctx, orderSN)
		if err != nil {
			return nil, "", "", err
		}
		return order, "trade_no", lookup.TradeNo, nil
	case lookup.CorrelationID != "":
		if err := query.Table("order_refund_requests").Select("order_sn").Where("correlation_id = ?", lookup.CorrelationID).Limit(1).Scan(&orderSN).Error; err != nil {
			return nil, "", "", wrapDatabaseError(err, "database operation")
		}
		if strings.TrimSpace(orderSN) == "" {
			return nil, "", "", errors.NewCode(bizcode.ErrOrderNotFound, "order not found")
		}
		order, err := o.Get(ctx, orderSN)
		if err != nil {
			return nil, "", "", err
		}
		return order, "correlation_id", lookup.CorrelationID, nil
	default:
		return nil, "", "", errors.NewCode(errcode.ErrValidation, "order_sn, trade_no or correlation_id is required")
	}
}

func (o *orders) loadTraceStatusLogs(ctx context.Context, orderSN string) ([]*do.OrderStatusLogDO, error) {
	entries := make([]*do.OrderStatusLogDO, 0)
	if err := o.db.WithContext(ctx).
		Where("order_sn = ?", orderSN).
		Order("add_time asc, id asc").
		Find(&entries).Error; err != nil {
		return nil, wrapDatabaseError(err, "database operation")
	}
	return entries, nil
}

func (o *orders) loadTraceRefunds(ctx context.Context, orderSN string) ([]do.RefundTraceRecord, error) {
	type refundTraceRow struct {
		RefundRequestID      uint64
		OrderSN              string
		ActorUserID          int32
		AmountFen            int64
		Reason               string
		Status               string
		Provider             string
		ProviderRefundID     string
		FailureReason        string
		CorrelationID        string
		RefundCreatedAt      time.Time
		RefundUpdatedAt      time.Time
		RefundJobID          sql.NullInt64
		RefundJobStatus      sql.NullString
		RefundJobAttempts    sql.NullInt64
		RefundJobAvailableAt sql.NullTime
		RefundJobLockedAt    sql.NullTime
		RefundJobLastError   sql.NullString
	}

	rows := make([]refundTraceRow, 0)
	if err := o.db.WithContext(ctx).
		Table("order_refund_requests AS r").
		Select(`r.id AS refund_request_id, r.order_sn, r.actor_user_id, r.amount_fen, r.reason, r.status, r.provider, r.provider_refund_id, r.failure_reason, r.correlation_id, r.created_at AS refund_created_at, r.updated_at AS refund_updated_at, o.id AS refund_job_id, o.status AS refund_job_status, o.attempts AS refund_job_attempts, o.available_at AS refund_job_available_at, o.locked_at AS refund_job_locked_at, o.last_error AS refund_job_last_error`).
		Joins("LEFT JOIN order_refund_outbox AS o ON o.refund_request_id = r.id").
		Where("r.order_sn = ?", orderSN).
		Order("r.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, wrapDatabaseError(err, "database operation")
	}

	refunds := make([]do.RefundTraceRecord, 0, len(rows))
	for _, row := range rows {
		record := do.RefundTraceRecord{
			RefundRequest: do.RefundRequestDO{
				ID:               row.RefundRequestID,
				OrderSN:          row.OrderSN,
				ActorUserID:      row.ActorUserID,
				AmountFen:        row.AmountFen,
				Reason:           row.Reason,
				Status:           row.Status,
				Provider:         row.Provider,
				ProviderRefundID: row.ProviderRefundID,
				FailureReason:    row.FailureReason,
				CorrelationID:    row.CorrelationID,
				CreatedAt:        row.RefundCreatedAt,
				UpdatedAt:        row.RefundUpdatedAt,
			},
		}
		if row.RefundJobID.Valid {
			outbox := &do.RefundOutboxDO{
				ID:              uint64(row.RefundJobID.Int64),
				RefundRequestID: row.RefundRequestID,
				Status:          row.RefundJobStatus.String,
				Attempts:        int(row.RefundJobAttempts.Int64),
				LastError:       row.RefundJobLastError.String,
			}
			if row.RefundJobAvailableAt.Valid {
				outbox.AvailableAt = row.RefundJobAvailableAt.Time
			}
			if row.RefundJobLockedAt.Valid {
				lockedAt := row.RefundJobLockedAt.Time
				outbox.LockedAt = &lockedAt
			}
			record.RefundOutbox = outbox
		}
		refunds = append(refunds, record)
	}
	return refunds, nil
}
