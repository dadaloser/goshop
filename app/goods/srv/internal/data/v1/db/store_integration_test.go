package db

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"goshop/app/goods/srv/internal/domain/do"
)

func TestGoodsStoreTransactionRollbackRealDB(t *testing.T) {
	db, _ := mustOpenSchemaIntegrationDB(t)
	prepareGoodsSchemaMigrations(t, db)
	store := newGoods(&mysqlFactory{db: db})
	outboxStore := newOutbox(&mysqlFactory{db: db})

	tx := store.Begin()
	if err := store.CreateInTxn(context.Background(), tx, testGoodsDO("rollback-sku")); err != nil {
		t.Fatalf("CreateInTxn(rollback goods) error = %v", err)
	}
	if err := outboxStore.CreateInTxn(context.Background(), tx, testOutboxEvent("rollback")); err != nil {
		t.Fatalf("CreateInTxn(rollback outbox) error = %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	var goodsCount, eventCount int64
	if err := db.Model(&do.GoodsDO{}).Count(&goodsCount).Error; err != nil {
		t.Fatalf("count rolled back goods error = %v", err)
	}
	if err := db.Model(&do.OutboxEventDO{}).Count(&eventCount).Error; err != nil {
		t.Fatalf("count rolled back events error = %v", err)
	}
	if goodsCount != 0 || eventCount != 0 {
		t.Errorf("rollback counts = goods:%d events:%d, want goods:0 events:0", goodsCount, eventCount)
	}

	persisted := testGoodsDO("crud-sku")
	if err := store.Create(context.Background(), persisted); err != nil {
		t.Fatalf("Create(goods=%q) error = %v", persisted.SKUCode, err)
	}
	if _, err := store.Get(context.Background(), uint64(persisted.ID)); err != nil {
		t.Fatalf("Get(goodsID=%d) error = %v", persisted.ID, err)
	}
	persisted.Name = "updated"
	if err := store.Update(context.Background(), persisted); err != nil {
		t.Fatalf("Update(goodsID=%d) error = %v", persisted.ID, err)
	}
	if count, err := store.CountByCategory(context.Background(), 1); err != nil || count != 1 {
		t.Errorf("CountByCategory(1) = %d, %v, want 1, nil", count, err)
	}
	if list, err := store.ListByIDs(context.Background(), []uint64{uint64(persisted.ID), uint64(persisted.ID)}, nil); err != nil || list.TotalCount != 1 {
		t.Errorf("ListByIDs([%d]) = total:%d, err:%v, want total:1, nil", persisted.ID, list.TotalCount, err)
	}
	if err := store.Delete(context.Background(), uint64(persisted.ID)); err != nil {
		t.Fatalf("Delete(goodsID=%d) error = %v", persisted.ID, err)
	}
}

func TestOutboxClaimConcurrentAndRecoveryRealDB(t *testing.T) {
	db, _ := mustOpenSchemaIntegrationDB(t)
	prepareGoodsSchemaMigrations(t, db)
	store := newOutbox(&mysqlFactory{db: db})
	now := time.Now().Unix()

	for i := 0; i < 20; i++ {
		if err := store.CreateInTxn(context.Background(), db, testOutboxEvent(fmt.Sprintf("claim-%d", i))); err != nil {
			t.Fatalf("CreateInTxn(event=%d) error = %v", i, err)
		}
	}

	start := make(chan struct{})
	results := make(chan []*do.OutboxEventDO, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for worker := 0; worker < 2; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			events, err := store.ClaimPending(context.Background(), do.OutboxTopicGoodsSync, 10, now)
			if err != nil {
				errs <- err
				return
			}
			results <- events
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Errorf("ClaimPending(concurrent) error = %v", err)
	}

	claimed := make(map[int32]struct{}, 20)
	for events := range results {
		for _, event := range events {
			if _, exists := claimed[event.ID]; exists {
				t.Errorf("ClaimPending(concurrent) duplicate event ID = %d", event.ID)
			}
			claimed[event.ID] = struct{}{}
		}
	}
	if got := len(claimed); got != 20 {
		t.Fatalf("ClaimPending(concurrent) claimed = %d, want 20", got)
	}
	if requeued, err := store.RequeueStale(context.Background(), do.OutboxTopicGoodsSync, now); err != nil {
		t.Fatalf("RequeueStale(claimedBefore=%d) error = %v", now, err)
	} else if requeued != 20 {
		t.Errorf("RequeueStale(claimedBefore=%d) = %d, want 20", now, requeued)
	}
	if pending, err := store.CountByStatus(context.Background(), do.OutboxTopicGoodsSync, do.OutboxStatusPending); err != nil {
		t.Fatalf("CountByStatus(PENDING) error = %v", err)
	} else if pending != 20 {
		t.Errorf("CountByStatus(PENDING) = %d, want 20", pending)
	}
}

func testGoodsDO(sku string) *do.GoodsDO {
	return &do.GoodsDO{CategoryID: 1, BrandsID: 1, Name: "data-layer", GoodsSn: sku, SPUCode: "spu-" + sku, SKUCode: sku, GoodsBrief: "brief", GoodsDesc: "desc", Images: do.GormList{}, DescImages: do.GormList{}, GoodsFrontImage: "image"}
}

func testOutboxEvent(key string) *do.OutboxEventDO {
	return &do.OutboxEventDO{Topic: do.OutboxTopicGoodsSync, AggregateType: "goods", AggregateID: 1, Action: do.OutboxActionUpsert, Payload: key, Status: do.OutboxStatusPending, NextAttemptAt: 0}
}
