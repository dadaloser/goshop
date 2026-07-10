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

	candidates, err := s.data.Orders().ListCloseCandidates(ctx, []string{OrderStatusWaitBuyerPay, OrderStatusPaying}, s.currentTime().Add(-s.lifecycle.TimeoutCloseAfter), s.lifecycle.BatchSize)
	if err != nil {
		return fmt.Errorf("list close candidates: %w", err)
	}

	orderSrv := newOrderService(s)
	for _, order := range candidates {
		if order == nil {
			continue
		}
		if err := s.releaseOrderInventory(ctx, order); err != nil {
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
			log.Errorf("close expired order %s: %v", order.OrderSn, err)
		}
	}
	return nil
}

func (s *service) processFinishedOrdersOnce(ctx context.Context) error {
	if s == nil || s.data == nil {
		return nil
	}

	candidates, err := s.data.Orders().ListFinishCandidates(ctx, OrderStatusTradeSuccess, s.currentTime().Add(-s.lifecycle.FinishAfterPayment), s.lifecycle.BatchSize)
	if err != nil {
		return fmt.Errorf("list finish candidates: %w", err)
	}

	orderSrv := newOrderService(s)
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
			log.Errorf("finish paid order %s: %v", order.OrderSn, err)
		}
	}
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
