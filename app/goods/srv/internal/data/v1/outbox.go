package v1

import (
	"context"

	"goshop/app/goods/srv/internal/domain/do"

	"gorm.io/gorm"
)

type OutboxStore interface {
	CreateInTxn(ctx context.Context, txn *gorm.DB, event *do.OutboxEventDO) error
	ClaimPending(ctx context.Context, topic string, limit int, nowUnix int64) ([]*do.OutboxEventDO, error)
	ListByStatus(ctx context.Context, topic, status string, limit int) ([]*do.OutboxEventDO, error)
	CountByStatus(ctx context.Context, topic, status string) (int64, error)
	RequeueStale(ctx context.Context, topic string, claimedBefore int64) (int64, error)
	MarkDone(ctx context.Context, id int32) error
	MarkRetry(ctx context.Context, id int32, retryCount int32, nextAttemptAt int64, lastError string) error
	MarkDead(ctx context.Context, id int32, retryCount int32, lastError string) error
	ReleaseClaim(ctx context.Context, id int32) error
}
