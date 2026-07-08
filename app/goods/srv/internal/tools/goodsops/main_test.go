package main

import (
	"context"
	"testing"

	"goshop/app/goods/srv/internal/domain/do"
	gorm2 "goshop/app/pkg/gorm"
)

func TestReplayDeadOutboxMarksDeadEventsPending(t *testing.T) {
	var retried []int32
	err := replayDeadOutbox(context.Background(), fakeOutboxOps{
		listByStatus: func(context.Context, string, string, int) ([]*do.OutboxEventDO, error) {
			return []*do.OutboxEventDO{
				{BaseModel: gorm2.BaseModel{ID: 1}, AggregateType: "goods"},
				{BaseModel: gorm2.BaseModel{ID: 2}, AggregateType: "goods"},
			}, nil
		},
		markRetry: func(_ context.Context, id int32, retryCount int32, nextAttemptAt int64, lastError string) error {
			retried = append(retried, id)
			return nil
		},
	}, 10)
	if err != nil {
		t.Fatalf("replayDeadOutbox() error = %v", err)
	}
	if len(retried) != 2 {
		t.Fatalf("replayDeadOutbox() retried = %d, want 2", len(retried))
	}
	if retried[0] != 1 || retried[1] != 2 {
		t.Fatalf("replayDeadOutbox() ids = %v, want [1 2]", retried)
	}
}

type fakeOutboxOps struct {
	listByStatus func(context.Context, string, string, int) ([]*do.OutboxEventDO, error)
	markRetry    func(context.Context, int32, int32, int64, string) error
}

func (f fakeOutboxOps) ListByStatus(ctx context.Context, topic, status string, limit int) ([]*do.OutboxEventDO, error) {
	return f.listByStatus(ctx, topic, status, limit)
}

func (f fakeOutboxOps) MarkRetry(ctx context.Context, id int32, retryCount int32, nextAttemptAt int64, lastError string) error {
	return f.markRetry(ctx, id, retryCount, nextAttemptAt, lastError)
}
