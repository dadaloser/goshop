package service

import (
	"context"
	"strings"
	"time"

	"goshop/app/order/srv/internal/domain/do"
	"goshop/gmicro/code"
	"goshop/pkg/errors"
)

type paymentEventStore interface {
	BeginPaymentEvent(context.Context, *do.PaymentEventDO) (*do.PaymentEventDO, *do.OrderInfoDO, bool, error)
	CompletePaymentEvent(context.Context, uint64, bool, string) error
	ListPaymentEvents(context.Context, string, int, int, bool) ([]do.PaymentEventDO, int64, int64, error)
}

type paymentWorkflowStore interface {
	ClaimRefundJobs(context.Context, int, int, time.Duration) ([]do.RefundJob, error)
	CompleteRefundJob(context.Context, uint64, bool, string, string, string, string, int) error
	ReconcilePayments(context.Context, string, time.Time, time.Time, []do.PaymentEventDO) (*do.PaymentReconciliationRunDO, error)
}

func (os *orderService) BeginPaymentEvent(ctx context.Context, event *do.PaymentEventDO) (*do.PaymentEventDO, *do.OrderInfoDO, bool, error) {
	store, ok := os.data.Orders().(paymentEventStore)
	if !ok {
		return nil, nil, false, errors.WithCode(code.ErrDatabase, "payment event store is not configured")
	}
	event.Provider = strings.ToLower(strings.TrimSpace(event.Provider))
	event.EventID = strings.TrimSpace(event.EventID)
	event.OrderSN = strings.TrimSpace(event.OrderSN)
	event.TradeNo = strings.TrimSpace(event.TradeNo)
	event.EventType = strings.ToLower(strings.TrimSpace(event.EventType))
	return store.BeginPaymentEvent(ctx, event)
}

func (os *orderService) ClaimRefundJobs(ctx context.Context, limit, maxAttempts int, lockTimeout time.Duration) ([]do.RefundJob, error) {
	store, ok := os.data.Orders().(paymentWorkflowStore)
	if !ok {
		return nil, errors.WithCode(code.ErrDatabase, "payment workflow store is not configured")
	}
	return store.ClaimRefundJobs(ctx, limit, maxAttempts, lockTimeout)
}

func (os *orderService) CompleteRefundJob(ctx context.Context, id uint64, success bool, provider, providerRefundID, providerStatus, detail string, maxAttempts int) error {
	store, ok := os.data.Orders().(paymentWorkflowStore)
	if !ok {
		return errors.WithCode(code.ErrDatabase, "payment workflow store is not configured")
	}
	return store.CompleteRefundJob(ctx, id, success, strings.ToLower(strings.TrimSpace(provider)), strings.TrimSpace(providerRefundID), strings.ToLower(strings.TrimSpace(providerStatus)), detail, maxAttempts)
}

func (os *orderService) ReconcilePayments(ctx context.Context, provider string, from, to time.Time, transactions []do.PaymentEventDO) (*do.PaymentReconciliationRunDO, error) {
	store, ok := os.data.Orders().(paymentWorkflowStore)
	if !ok {
		return nil, errors.WithCode(code.ErrDatabase, "payment workflow store is not configured")
	}
	return store.ReconcilePayments(ctx, provider, from, to, transactions)
}
func (os *orderService) CompletePaymentEvent(ctx context.Context, id uint64, success bool, detail string) error {
	store, ok := os.data.Orders().(paymentEventStore)
	if !ok {
		return errors.WithCode(code.ErrDatabase, "payment event store is not configured")
	}
	return store.CompletePaymentEvent(ctx, id, success, detail)
}
func (os *orderService) ListPaymentEvents(ctx context.Context, orderSN string, page, pageSize int, mismatchesOnly bool) ([]do.PaymentEventDO, int64, int64, error) {
	store, ok := os.data.Orders().(paymentEventStore)
	if !ok {
		return nil, 0, 0, errors.WithCode(code.ErrDatabase, "payment event store is not configured")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return store.ListPaymentEvents(ctx, strings.TrimSpace(orderSN), (page-1)*pageSize, pageSize, mismatchesOnly)
}
