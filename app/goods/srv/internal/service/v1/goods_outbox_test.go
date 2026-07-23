package v1

import (
	"context"
	"testing"

	"goshop/app/goods/srv/internal/domain/do"
	gorm2 "goshop/app/pkg/gorm"
)

func TestProcessGoodsOutboxEventMarksDoneOnSuccess(t *testing.T) {
	doneID := int32(0)
	err := processGoodsOutboxEvent(
		context.Background(),
		fakeOutboxStore{
			markDone: func(_ context.Context, id int32) error {
				doneID = id
				return nil
			},
		},
		fakeSearchGoodsStore{},
		&do.OutboxEventDO{
			BaseModel: gorm2.BaseModel{ID: 1},
			Topic:     do.OutboxTopicGoodsSync,
			Action:    do.OutboxActionUpsert,
			Payload:   `{"action":"UPSERT","id":1,"goods":{"id":1,"name":"goods"}}`,
			Status:    do.OutboxStatusProcessing,
		},
	)
	if err != nil {
		t.Fatalf("processGoodsOutboxEvent() error = %v", err)
	}
	if doneID != 0 {
		return
	}
	t.Fatal("processGoodsOutboxEvent() did not mark done")
}

func TestProcessGoodsOutboxEventMarksRetryOnFailure(t *testing.T) {
	retried := false
	err := processGoodsOutboxEvent(
		context.Background(),
		fakeOutboxStore{
			markRetry: func(_ context.Context, id int32, retryCount int32, nextAttemptAt int64, lastError string) error {
				retried = id == 7 && retryCount == 2 && nextAttemptAt > 0 && lastError != ""
				return nil
			},
		},
		fakeSearchGoodsStore{
			update: func(context.Context, *do.GoodsSearchDO) error {
				return assertErr("boom")
			},
		},
		&do.OutboxEventDO{
			BaseModel:     gorm2.BaseModel{ID: 7},
			Action:        do.OutboxActionUpsert,
			Payload:       `{"action":"UPSERT","id":7,"goods":{"id":7,"name":"goods"}}`,
			Status:        do.OutboxStatusProcessing,
			RetryCount:    1,
			MaxRetryCount: 5,
		},
	)
	if err != nil {
		t.Fatalf("processGoodsOutboxEvent() error = %v", err)
	}
	if !retried {
		t.Fatal("processGoodsOutboxEvent() did not mark retry")
	}
}

func TestProcessGoodsOutboxEventMarksDeadAfterMaxRetry(t *testing.T) {
	dead := false
	err := processGoodsOutboxEvent(
		context.Background(),
		fakeOutboxStore{
			markDead: func(_ context.Context, id int32, retryCount int32, lastError string) error {
				dead = id == 8 && retryCount == 5 && lastError != ""
				return nil
			},
		},
		fakeSearchGoodsStore{
			delete: func(context.Context, uint64) error {
				return assertErr("boom")
			},
		},
		&do.OutboxEventDO{
			BaseModel:     gorm2.BaseModel{ID: 8},
			Action:        do.OutboxActionDelete,
			Payload:       `{"action":"DELETE","id":8}`,
			Status:        do.OutboxStatusProcessing,
			RetryCount:    4,
			MaxRetryCount: 5,
		},
	)
	if err != nil {
		t.Fatalf("processGoodsOutboxEvent() error = %v", err)
	}
	if !dead {
		t.Fatal("processGoodsOutboxEvent() did not mark dead")
	}
}

func TestProcessGoodsOutboxEventUsesCurrentMySQLState(t *testing.T) {
	var indexed *do.GoodsSearchDO
	err := processGoodsOutboxEvent(
		context.Background(),
		fakeOutboxStore{markDone: func(context.Context, int32) error { return nil }},
		fakeSearchGoodsStore{update: func(_ context.Context, goods *do.GoodsSearchDO) error {
			indexed = goods
			return nil
		}},
		&do.OutboxEventDO{
			BaseModel:   gorm2.BaseModel{ID: 9},
			AggregateID: 7,
			Action:      do.OutboxActionUpsert,
			Payload:     `{"action":"UPSERT","id":7,"goods":{"id":7,"name":"stale"}}`,
		},
		func(context.Context, uint64) (*do.GoodsSearchDO, error) {
			return &do.GoodsSearchDO{ID: 7, Name: "current", OnSale: true}, nil
		},
	)
	if err != nil {
		t.Fatalf("processGoodsOutboxEvent() error = %v", err)
	}
	if indexed == nil || indexed.Name != "current" || !indexed.OnSale {
		t.Fatalf("indexed goods = %+v, want current MySQL state", indexed)
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
