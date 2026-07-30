package v1

import (
	"context"
	"time"
)

// AccountDeletionOutboxStore persists and leases pending deletion events.
type AccountDeletionOutboxStore interface {
	ClaimPendingDeletionEvents(ctx context.Context, limit int, now time.Time) ([]*AccountDeletionOutboxEventDO, error)
	MarkDeletionEventPublished(ctx context.Context, eventID string, at time.Time) error
	RetryDeletionEvent(ctx context.Context, eventID string, retryCount int, availableAt time.Time, lastError string) error
	RequeueStaleDeletionEvents(ctx context.Context, before time.Time) (int64, error)
}
