package v1

import (
	"context"
	"fmt"

	datav1 "goshop/app/goods/srv/internal/data/v1"
	"goshop/app/goods/srv/internal/domain/do"
	"goshop/gmicro/core/metric"
)

var (
	metricGoodsOutboxBacklog        = metric.NewGaugeVec(&metric.GaugeVecOpts{Namespace: "goods_service", Subsystem: "outbox", Name: "backlog", Help: "Current goods outbox events grouped by status.", Labels: []string{"topic", "status"}})
	metricGoodsOutboxProcessedTotal = metric.NewCounterVec(&metric.CounterVecOpts{Namespace: "goods_service", Subsystem: "outbox", Name: "processed_total", Help: "Goods outbox processing outcomes.", Labels: []string{"action", "result"}})
	metricGoodsOutboxRecoveredTotal = metric.NewCounterVec(&metric.CounterVecOpts{Namespace: "goods_service", Subsystem: "outbox", Name: "recovered_total", Help: "Stale processing claims recovered by topic.", Labels: []string{"topic"}})
)

func observeGoodsOutboxBacklog(ctx context.Context, store datav1.OutboxStore) error {
	for _, status := range []string{do.OutboxStatusPending, do.OutboxStatusProcessing, do.OutboxStatusDead} {
		count, err := store.CountByStatus(ctx, do.OutboxTopicGoodsSync, status)
		if err != nil {
			return fmt.Errorf("count goods outbox %s backlog: %w", status, err)
		}
		metricGoodsOutboxBacklog.Set(float64(count), do.OutboxTopicGoodsSync, status)
	}
	return nil
}
