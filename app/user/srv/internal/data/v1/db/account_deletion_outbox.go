package db

import (
	"context"
	"strings"
	"time"

	dv1 "goshop/app/user/srv/internal/data/v1"
	"goshop/pkg/errcode"
	"goshop/pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type accountDeletionOutbox struct{ db *gorm.DB }

func NewAccountDeletionOutboxStore(db *gorm.DB) dv1.AccountDeletionOutboxStore {
	return &accountDeletionOutbox{db: db}
}

func (s *accountDeletionOutbox) ClaimPendingDeletionEvents(ctx context.Context, limit int, now time.Time) ([]*dv1.AccountDeletionOutboxEventDO, error) {
	if limit <= 0 {
		limit = 50
	}
	events := make([]*dv1.AccountDeletionOutboxEventDO, 0, limit)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND available_at <= ?", "PENDING", now.UTC()).
			Order("available_at ASC, created_at ASC").Limit(limit).Find(&events).Error; err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		ids := make([]string, 0, len(events))
		for _, event := range events {
			if event != nil {
				ids = append(ids, event.ID)
			}
		}
		return tx.Model(&dv1.AccountDeletionOutboxEventDO{}).Where("id IN ? AND status = ?", ids, "PENDING").Updates(map[string]interface{}{"status": "PROCESSING", "locked_at": now.UTC()}).Error
	})
	if err != nil {
		return nil, errors.NewCode(errcode.ErrDatabase, err.Error())
	}
	return events, nil
}

func (s *accountDeletionOutbox) MarkDeletionEventPublished(ctx context.Context, eventID string, at time.Time) error {
	return s.update(ctx, eventID, map[string]interface{}{"status": "PUBLISHED", "published_at": at.UTC(), "locked_at": nil, "last_error": ""})
}

func (s *accountDeletionOutbox) RetryDeletionEvent(ctx context.Context, eventID string, retryCount int, availableAt time.Time, lastError string) error {
	return s.update(ctx, eventID, map[string]interface{}{"status": "PENDING", "retry_count": retryCount, "available_at": availableAt.UTC(), "locked_at": nil, "last_error": trimDeletionOutboxError(lastError)})
}

func (s *accountDeletionOutbox) RequeueStaleDeletionEvents(ctx context.Context, before time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Model(&dv1.AccountDeletionOutboxEventDO{}).Where("status = ? AND locked_at <= ?", "PROCESSING", before.UTC()).Updates(map[string]interface{}{"status": "PENDING", "locked_at": nil})
	if result.Error != nil {
		return 0, errors.NewCode(errcode.ErrDatabase, result.Error.Error())
	}
	return result.RowsAffected, nil
}

func (s *accountDeletionOutbox) update(ctx context.Context, eventID string, values map[string]interface{}) error {
	if strings.TrimSpace(eventID) == "" {
		return errors.NewCode(errcode.ErrValidation, "account deletion event id is required")
	}
	if err := s.db.WithContext(ctx).Model(&dv1.AccountDeletionOutboxEventDO{}).Where("id = ?", eventID).Updates(values).Error; err != nil {
		return errors.NewCode(errcode.ErrDatabase, err.Error())
	}
	return nil
}

func trimDeletionOutboxError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

var _ dv1.AccountDeletionOutboxStore = &accountDeletionOutbox{}
