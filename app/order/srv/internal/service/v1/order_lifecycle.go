package service

import (
	"context"
	"fmt"
	"goshop/app/order/srv/internal/boundary"
	"goshop/app/order/srv/internal/domain/do"
	"goshop/app/order/srv/internal/domain/dto"
	"goshop/pkg/log"
	"strings"
	"time"
)

const (
	orderLifecyclePollInterval = 5 * time.Second
	orderTimeoutCloseAfter     = 30 * time.Minute
	orderFinishAfterPayment    = 0
	orderLifecycleBatchSize    = 20
)

func (s *service) runLifecycleWorker(ctx context.Context) error {
	ticker := time.NewTicker(s.lifecycle.PollInterval)
	defer ticker.Stop()

	s.runLifecycleSweep(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.runLifecycleSweep(ctx)
		}
	}
}

func (s *service) runLifecycleSweep(ctx context.Context) {
	if err := s.processExpiredOrdersOnce(ctx); err != nil {
		log.Errorf("process expired orders: %v", err)
	}
	if err := s.processFinishedOrdersOnce(ctx); err != nil {
		log.Errorf("process finished paid orders: %v", err)
	}
}

func (s *service) processExpiredOrdersOnce(ctx context.Context) error {
	if s == nil || s.data == nil {
		return nil
	}
	startedAt := time.Now()
	result := "success"
	defer func() {
		observeLifecycleSweep("auto_close", result, startedAt)
	}()

	candidates, err := s.data.Orders().ListCloseCandidates(ctx, []string{OrderStatusWaitBuyerPay, OrderStatusPaying}, s.currentTime().Add(-s.lifecycle.TimeoutCloseAfter), s.lifecycle.BatchSize)
	if err != nil {
		result = "failed"
		return fmt.Errorf("list close candidates: %w", err)
	}
	metricOrderLifecycleCandidatesTotal.Add(float64(len(candidates)), "auto_close")

	orderSrv := newOrderService(s)
	var closedCount int
	var failedCount int
	for _, order := range candidates {
		if order == nil {
			continue
		}
		if err := s.releaseOrderInventory(ctx, order); err != nil {
			failedCount++
			metricOrderLifecycleProcessedTotal.Inc("auto_close", "failed")
			log.Errorf("release inventory for expired order %s: %v", order.OrderSn, err)
			continue
		}
		if err := orderSrv.Update(ctx, &dto.OrderDTO{
			OrderInfoDO: do.OrderInfoDO{
				OrderSn: order.OrderSn,
				Status:  OrderStatusTradeClosed,
			},
			StatusReason:   "order timeout auto close",
			StatusSource:   "order.timeout_worker",
			StatusOperator: "system",
		}); err != nil {
			failedCount++
			metricOrderLifecycleProcessedTotal.Inc("auto_close", "failed")
			log.Errorf("close expired order %s: %v", order.OrderSn, err)
			continue
		}
		closedCount++
		metricOrderLifecycleProcessedTotal.Inc("auto_close", "success")
	}
	if failedCount > 0 {
		result = "failed"
	}
	log.InfoC(ctx, "order lifecycle auto close sweep completed",
		log.Int("candidates", len(candidates)),
		log.Int("closed", closedCount),
		log.Int("failed", failedCount),
		log.Duration("timeout_close_after", s.lifecycle.TimeoutCloseAfter),
		log.Int("batch_size", s.lifecycle.BatchSize),
		log.Duration("duration", time.Since(startedAt)),
	)
	return nil
}

func (s *service) processFinishedOrdersOnce(ctx context.Context) error {
	if s == nil || s.data == nil {
		return nil
	}
	startedAt := time.Now()
	result := "success"
	defer func() {
		observeLifecycleSweep("auto_finish", result, startedAt)
	}()

	candidates, err := s.data.Orders().ListFinishCandidates(ctx, OrderStatusTradeSuccess, s.currentTime().Add(-s.lifecycle.FinishAfterPayment), s.lifecycle.BatchSize)
	if err != nil {
		result = "failed"
		return fmt.Errorf("list finish candidates: %w", err)
	}
	metricOrderLifecycleCandidatesTotal.Add(float64(len(candidates)), "auto_finish")

	orderSrv := newOrderService(s)
	var finishedCount int
	var failedCount int
	for _, order := range candidates {
		if order == nil {
			continue
		}
		if err := orderSrv.Update(ctx, &dto.OrderDTO{
			OrderInfoDO: do.OrderInfoDO{
				OrderSn: order.OrderSn,
				Status:  OrderStatusTradeFinished,
			},
			StatusReason:   "payment success auto finish",
			StatusSource:   "order.finish_worker",
			StatusOperator: "system",
		}); err != nil {
			failedCount++
			metricOrderLifecycleProcessedTotal.Inc("auto_finish", "failed")
			log.Errorf("finish paid order %s: %v", order.OrderSn, err)
			continue
		}
		finishedCount++
		metricOrderLifecycleProcessedTotal.Inc("auto_finish", "success")
	}
	if failedCount > 0 {
		result = "failed"
	}
	log.InfoC(ctx, "order lifecycle auto finish sweep completed",
		log.Int("candidates", len(candidates)),
		log.Int("finished", finishedCount),
		log.Int("failed", failedCount),
		log.Duration("finish_after_payment", s.lifecycle.FinishAfterPayment),
		log.Int("batch_size", s.lifecycle.BatchSize),
		log.Duration("duration", time.Since(startedAt)),
	)
	return nil
}

func (s *service) releaseOrderInventory(ctx context.Context, order *do.OrderInfoDO) error {
	if s == nil || s.upstream.inventory == nil || order == nil {
		return nil
	}

	items := make([]boundary.InventoryItem, 0, len(order.OrderGoods))
	for _, item := range order.OrderGoods {
		if item == nil || item.Goods <= 0 || item.Nums <= 0 {
			continue
		}
		items = append(items, boundary.InventoryItem{
			GoodsID: item.Goods,
			Num:     item.Nums,
		})
	}
	if len(items) == 0 {
		return nil
	}
	return s.upstream.inventory.Release(ctx, strings.TrimSpace(order.OrderSn), items)
}

func (s *service) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}
