package db

import (
	"context"
	"strings"

	v1 "goshop/app/goods/srv/internal/data/v1"
	"goshop/app/goods/srv/internal/domain/do"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"gorm.io/gorm"
)

type outbox struct {
	db *gorm.DB
}

func newOutbox(factory *mysqlFactory) *outbox {
	return &outbox{db: factory.db}
}

func (o *outbox) CreateInTxn(ctx context.Context, txn *gorm.DB, event *do.OutboxEventDO) error {
	if txn == nil || event == nil {
		return errors.WithCode(code2.ErrValidation, "outbox event is required")
	}
	if err := txn.WithContext(ctx).Create(event).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func (o *outbox) ClaimPending(ctx context.Context, topic string, limit int, nowUnix int64) ([]*do.OutboxEventDO, error) {
	if limit <= 0 {
		limit = 10
	}

	var events []*do.OutboxEventDO
	err := o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("topic = ? AND status = ? AND next_attempt_at <= ?", topic, do.OutboxStatusPending, nowUnix).
			Order("id asc").
			Limit(limit).
			Find(&events).Error; err != nil {
			return errors.WithCode(code2.ErrDatabase, err.Error())
		}
		if len(events) == 0 {
			return nil
		}

		ids := make([]int32, 0, len(events))
		for _, event := range events {
			if event == nil {
				continue
			}
			ids = append(ids, event.ID)
		}
		if len(ids) == 0 {
			events = nil
			return nil
		}

		if err := tx.Model(&do.OutboxEventDO{}).
			Where("id IN ? AND status = ?", ids, do.OutboxStatusPending).
			Updates(map[string]interface{}{
				"status":          do.OutboxStatusProcessing,
				"processing_lock": topic,
			}).Error; err != nil {
			return errors.WithCode(code2.ErrDatabase, err.Error())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (o *outbox) ListByStatus(ctx context.Context, topic, status string, limit int) ([]*do.OutboxEventDO, error) {
	if limit <= 0 {
		limit = 100
	}

	var events []*do.OutboxEventDO
	query := o.db.WithContext(ctx).Order("id asc").Limit(limit)
	if strings.TrimSpace(topic) != "" {
		query = query.Where("topic = ?", topic)
	}
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&events).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return events, nil
}

func (o *outbox) MarkDone(ctx context.Context, id int32) error {
	return o.updateStatus(ctx, id, map[string]interface{}{
		"status":          do.OutboxStatusDone,
		"last_error":      "",
		"processing_lock": "",
	})
}

func (o *outbox) MarkRetry(ctx context.Context, id int32, retryCount int32, nextAttemptAt int64, lastError string) error {
	return o.updateStatus(ctx, id, map[string]interface{}{
		"status":          do.OutboxStatusPending,
		"retry_count":     retryCount,
		"next_attempt_at": nextAttemptAt,
		"last_error":      trimError(lastError),
		"processing_lock": "",
	})
}

func (o *outbox) MarkDead(ctx context.Context, id int32, retryCount int32, lastError string) error {
	return o.updateStatus(ctx, id, map[string]interface{}{
		"status":          do.OutboxStatusDead,
		"retry_count":     retryCount,
		"last_error":      trimError(lastError),
		"processing_lock": "",
	})
}

func (o *outbox) ReleaseClaim(ctx context.Context, id int32) error {
	return o.updateStatus(ctx, id, map[string]interface{}{
		"status":          do.OutboxStatusPending,
		"processing_lock": "",
	})
}

func (o *outbox) updateStatus(ctx context.Context, id int32, updates map[string]interface{}) error {
	if id <= 0 {
		return errors.WithCode(code2.ErrValidation, "outbox event id is required")
	}
	if err := o.db.WithContext(ctx).
		Model(&do.OutboxEventDO{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func trimError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 500 {
		return value
	}
	return value[:500]
}

var _ v1.OutboxStore = &outbox{}
