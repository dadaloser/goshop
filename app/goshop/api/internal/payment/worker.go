package payment

import (
	"context"
	"fmt"
	"time"

	opb "goshop/api/order/v1"
	"goshop/app/pkg/options"
	"goshop/pkg/log"

	"golang.org/x/sync/errgroup"
)

type Worker struct {
	orders   opb.OrderClient
	provider Provider
	opts     *options.PaymentOptions
	now      func() time.Time
}

func NewWorker(orders opb.OrderClient, provider Provider, opts *options.PaymentOptions) *Worker {
	return &Worker{orders: orders, provider: provider, opts: opts, now: time.Now}
}

func (w *Worker) Run(ctx context.Context) error {
	if w == nil || w.orders == nil || w.provider == nil || w.opts == nil || !w.opts.Enabled {
		return nil
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error { return w.runRefunds(groupCtx) })
	group.Go(func() error { return w.runReconciliation(groupCtx) })
	return group.Wait()
}

func (w *Worker) runRefunds(ctx context.Context) error {
	ticker := time.NewTicker(w.opts.WorkerInterval)
	defer ticker.Stop()
	for {
		if err := w.processRefundBatch(ctx); err != nil && ctx.Err() == nil {
			log.Errorf("process payment refund batch: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (w *Worker) processRefundBatch(ctx context.Context) error {
	claim, err := w.orders.ClaimRefundJobs(ctx, &opb.ClaimRefundJobsRequest{Limit: int32(w.opts.WorkerBatchSize), MaxAttempts: int32(w.opts.MaxAttempts), LockTimeoutSeconds: 120})
	if err != nil {
		return fmt.Errorf("claim refund jobs: %w", err)
	}
	for _, job := range claim.GetJobs() {
		requestCtx, cancel := context.WithTimeout(ctx, w.opts.RequestTimeout)
		response, refundErr := w.provider.Refund(requestCtx, RefundRequest{RequestID: job.GetCorrelationId(), OrderSN: job.GetOrderSn(), TradeNo: job.GetTradeNo(), AmountFen: job.GetAmountFen(), Reason: job.GetReason()})
		cancel()
		complete := &opb.CompleteRefundJobRequest{Id: job.GetId(), Success: refundErr == nil, Provider: w.opts.Provider, ProviderRefundId: response.ProviderRefundID, ProviderStatus: response.Status, MaxAttempts: int32(w.opts.MaxAttempts)}
		if refundErr != nil {
			complete.ErrorDetail = refundErr.Error()
		}
		if _, err := w.orders.CompleteRefundJob(ctx, complete); err != nil {
			return fmt.Errorf("complete refund job %d: %w", job.GetId(), err)
		}
	}
	return nil
}

func (w *Worker) runReconciliation(ctx context.Context) error {
	interval := time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := w.reconcileWindow(ctx, w.now().Add(-interval), w.now()); err != nil && ctx.Err() == nil {
			log.Errorf("reconcile payment window: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (w *Worker) reconcileWindow(ctx context.Context, from, to time.Time) error {
	requestCtx, cancel := context.WithTimeout(ctx, w.opts.RequestTimeout)
	transactions, err := w.provider.ListTransactions(requestCtx, from, to)
	cancel()
	if err != nil {
		return fmt.Errorf("load provider reconciliation statement: %w", err)
	}
	items := make([]*opb.ProviderTransaction, 0, len(transactions))
	for _, transaction := range transactions {
		items = append(items, &opb.ProviderTransaction{EventId: transaction.EventID, OrderSn: transaction.OrderSN, TradeNo: transaction.TradeNo, EventType: transaction.EventType, AmountFen: transaction.AmountFen, OccurredAt: transaction.OccurredAt.Unix()})
	}
	if _, err := w.orders.ReconcilePayments(ctx, &opb.ReconcilePaymentsRequest{Provider: w.opts.Provider, WindowStart: from.Unix(), WindowEnd: to.Unix(), Transactions: items}); err != nil {
		return fmt.Errorf("persist payment reconciliation: %w", err)
	}
	return nil
}
