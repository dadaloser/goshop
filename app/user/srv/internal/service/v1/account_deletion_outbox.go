package v1

import (
	"context"
	"fmt"
	"time"

	"goshop/app/pkg/eventbus"
	dv1 "goshop/app/user/srv/internal/data/v1"
	"goshop/pkg/log"
)

const accountDeletionOutboxStateWriteTimeout = 5 * time.Second

type AccountDeletionOutboxConfig struct {
	NATSURL        string
	PollInterval   time.Duration
	SweepTimeout   time.Duration
	PublishTimeout time.Duration
	BatchSize      int
	MaxRetries     int
}

func (c AccountDeletionOutboxConfig) normalized() AccountDeletionOutboxConfig {
	if c.PollInterval <= 0 {
		c.PollInterval = 2 * time.Second
	}
	if c.SweepTimeout <= 0 {
		c.SweepTimeout = 2 * time.Minute
	}
	if c.PublishTimeout <= 0 {
		c.PublishTimeout = 5 * time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 20
	}
	return c
}

type AccountDeletionOutboxWorker struct {
	store dv1.AccountDeletionOutboxStore
	cfg   AccountDeletionOutboxConfig
}

func NewAccountDeletionOutboxWorker(store dv1.AccountDeletionOutboxStore, cfg AccountDeletionOutboxConfig) *AccountDeletionOutboxWorker {
	return &AccountDeletionOutboxWorker{store: store, cfg: cfg.normalized()}
}

func (w *AccountDeletionOutboxWorker) Run(ctx context.Context) error {
	if w == nil || w.store == nil || w.cfg.NATSURL == "" {
		return nil
	}
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		sweepCtx, cancel := context.WithTimeout(ctx, w.cfg.SweepTimeout)
		w.process(sweepCtx, ctx)
		cancel()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *AccountDeletionOutboxWorker) process(ctx, stateParent context.Context) {
	now := time.Now().UTC()
	if _, err := w.store.RequeueStaleDeletionEvents(ctx, now.Add(-5*time.Minute)); err != nil {
		log.Errorf("requeue stale account deletion events: %v", err)
		return
	}
	publisher, err := eventbus.Connect(eventbus.Config{URL: w.cfg.NATSURL, PublishTimeout: w.cfg.PublishTimeout})
	if err != nil {
		log.Warnf("account deletion event bus unavailable: %v", err)
		return
	}
	defer publisher.Close()
	events, err := w.store.ClaimPendingDeletionEvents(ctx, w.cfg.BatchSize, now)
	if err != nil {
		log.Errorf("claim account deletion events: %v", err)
		return
	}
	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return
		}
		if event == nil {
			continue
		}
		err = publisher.Publish(ctx, eventbus.Event{ID: event.ID, Subject: event.EventType, OccurredAt: event.CreatedAt, Payload: event.Payload, CorrelationID: event.ID})
		if err == nil {
			stateCtx, cancel := accountDeletionOutboxStateContext(stateParent)
			err = w.store.MarkDeletionEventPublished(stateCtx, event.ID, time.Now().UTC())
			cancel()
		}
		if err == nil {
			continue
		}
		retry := event.RetryCount + 1
		if retry > w.cfg.MaxRetries {
			retry = w.cfg.MaxRetries
		}
		delay := time.Duration(1<<min(retry, 8)) * time.Second
		if retry == w.cfg.MaxRetries {
			delay = time.Hour
		}
		stateCtx, cancel := accountDeletionOutboxStateContext(stateParent)
		retryErr := w.store.RetryDeletionEvent(stateCtx, event.ID, retry, time.Now().UTC().Add(delay), fmt.Sprintf("%v", err))
		cancel()
		if retryErr != nil {
			log.Errorf("retry account deletion event %s: %v", event.ID, retryErr)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func accountDeletionOutboxStateContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, accountDeletionOutboxStateWriteTimeout)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
