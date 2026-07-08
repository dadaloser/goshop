package v1

import (
	"context"
	"encoding/json"
	"fmt"
	datav1 "goshop/app/goods/srv/internal/data/v1"
	"goshop/app/goods/srv/internal/domain/do"
	"goshop/app/goods/srv/internal/domain/dto"
	"time"
)

const (
	outboxPollInterval = 2 * time.Second
	outboxBatchSize    = 20
	outboxMaxRetry     = 5
)

type goodsOutboxPayload struct {
	Action string            `json:"action"`
	Goods  *do.GoodsSearchDO `json:"goods,omitempty"`
	ID     uint64            `json:"id,omitempty"`
}

func newGoodsSyncEvent(goods *dto.GoodsDTO) (*do.OutboxEventDO, error) {
	searchDO := goodsSearchFromDTO(goods)
	payload, err := json.Marshal(goodsOutboxPayload{
		Action: do.OutboxActionUpsert,
		Goods:  &searchDO,
		ID:     uint64(goods.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal goods outbox payload: %w", err)
	}
	return &do.OutboxEventDO{
		Topic:         do.OutboxTopicGoodsSync,
		AggregateType: "goods",
		AggregateID:   goods.ID,
		Action:        do.OutboxActionUpsert,
		Payload:       string(payload),
		Status:        do.OutboxStatusPending,
		MaxRetryCount: outboxMaxRetry,
	}, nil
}

func newGoodsDeleteEvent(id uint64) (*do.OutboxEventDO, error) {
	payload, err := json.Marshal(goodsOutboxPayload{
		Action: do.OutboxActionDelete,
		ID:     id,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal goods delete outbox payload: %w", err)
	}
	return &do.OutboxEventDO{
		Topic:         do.OutboxTopicGoodsSync,
		AggregateType: "goods",
		AggregateID:   int32(id),
		Action:        do.OutboxActionDelete,
		Payload:       string(payload),
		Status:        do.OutboxStatusPending,
		MaxRetryCount: outboxMaxRetry,
	}, nil
}

func (s *service) runGoodsOutboxWorker(ctx context.Context) error {
	ticker := time.NewTicker(outboxPollInterval)
	defer ticker.Stop()

	if err := s.processGoodsOutboxOnce(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.processGoodsOutboxOnce(ctx); err != nil {
				return err
			}
		}
	}
}

func (s *service) processGoodsOutboxOnce(ctx context.Context) error {
	if s == nil || s.data == nil || s.dataSearch == nil {
		return nil
	}
	outboxStore := s.data.Outbox()
	if outboxStore == nil {
		return nil
	}
	searchGoods := s.dataSearch.Goods()
	if searchGoods == nil {
		return nil
	}

	events, err := outboxStore.ClaimPending(ctx, do.OutboxTopicGoodsSync, outboxBatchSize, time.Now().Unix())
	if err != nil {
		return err
	}
	for _, event := range events {
		if event == nil {
			continue
		}
		if err := processGoodsOutboxEvent(ctx, outboxStore, searchGoods, event); err != nil {
			return err
		}
	}
	return nil
}

func processGoodsOutboxEvent(ctx context.Context, outboxStore datav1.OutboxStore, searchGoods searchGoodsStore, event *do.OutboxEventDO) error {
	payload := goodsOutboxPayload{}
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		return outboxStore.MarkDead(ctx, event.ID, event.RetryCount+1, err.Error())
	}

	var processErr error
	switch payload.Action {
	case do.OutboxActionUpsert:
		if payload.Goods == nil || payload.Goods.ID <= 0 {
			processErr = fmt.Errorf("invalid goods upsert payload")
		} else {
			processErr = searchGoods.Update(ctx, payload.Goods)
		}
	case do.OutboxActionDelete:
		if payload.ID == 0 {
			processErr = fmt.Errorf("invalid goods delete payload")
		} else {
			processErr = searchGoods.Delete(ctx, payload.ID)
		}
	default:
		processErr = fmt.Errorf("unsupported outbox action %q", payload.Action)
	}
	if processErr == nil {
		return outboxStore.MarkDone(ctx, event.ID)
	}

	retryCount := event.RetryCount + 1
	if retryCount >= event.MaxRetryCount {
		return outboxStore.MarkDead(ctx, event.ID, retryCount, processErr.Error())
	}

	nextAttempt := time.Now().Add(backoffDuration(retryCount)).Unix()
	return outboxStore.MarkRetry(ctx, event.ID, retryCount, nextAttempt, processErr.Error())
}

func backoffDuration(retryCount int32) time.Duration {
	if retryCount <= 0 {
		return time.Second
	}
	return time.Duration(retryCount) * time.Second
}

type searchGoodsStore interface {
	Update(ctx context.Context, goods *do.GoodsSearchDO) error
	Delete(ctx context.Context, id uint64) error
}
