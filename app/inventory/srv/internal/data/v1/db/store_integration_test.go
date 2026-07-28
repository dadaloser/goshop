package mysql

import (
	"context"
	"fmt"
	"goshop/app/pkg/bizcode"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/pkg/errors"
)

func TestInventoryStoreRollbackAndAdjustmentRecoveryRealDB(t *testing.T) {
	db, _ := mustOpenSchemaIntegrationDB(t)
	prepareInventorySchemaMigrations(t, db)
	store := newInventorys(&mysqlStore{db: db})
	goodsID := int32(time.Now().UnixNano() % 1_000_000_000)
	if err := store.Create(context.Background(), &do.InventoryDO{Goods: goodsID, Stocks: 10}); err != nil {
		t.Fatalf("Create(goodsID=%d) error = %v", goodsID, err)
	}

	tx := db.Begin()
	if err := store.Reduce(context.Background(), tx, uint64(goodsID), 4); err != nil {
		t.Fatalf("Reduce(transaction goodsID=%d) error = %v", goodsID, err)
	}
	if err := store.CreateStockSellDetail(context.Background(), tx, &do.StockSellDetailDO{OrderSn: fmt.Sprintf("rollback-%d", goodsID), Status: 1, Detail: do.GoodsDetailList{{Goods: goodsID, Num: 4}}}); err != nil {
		t.Fatalf("CreateStockSellDetail(transaction goodsID=%d) error = %v", goodsID, err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("Rollback(goodsID=%d) error = %v", goodsID, err)
	}
	assertInventoryAvailable(t, store, goodsID, 10, 0, 0)

	first := &do.InventoryDO{Goods: goodsID, Available: 8, Total: 10, Locked: 0, Sold: 0}
	audit := &do.InventoryAdjustmentDO{ActorUserID: 1, CorrelationID: fmt.Sprintf("adjust-%d", goodsID), RequestID: "request-1", Reason: "recovery test", CreatedAt: time.Now()}
	if err := store.Adjust(context.Background(), first, audit); err != nil {
		t.Fatalf("Adjust(first goodsID=%d) error = %v", goodsID, err)
	}
	second := &do.InventoryDO{Goods: goodsID, Available: 3, Total: 10, Locked: 0, Sold: 0}
	duplicateAudit := &do.InventoryAdjustmentDO{ActorUserID: 1, CorrelationID: audit.CorrelationID, RequestID: "request-2", Reason: "duplicate audit", CreatedAt: time.Now()}
	if err := store.Adjust(context.Background(), second, duplicateAudit); err == nil {
		t.Fatal("Adjust(duplicate correlation) error = nil, want transaction failure")
	}
	assertInventoryAvailable(t, store, goodsID, 8, 0, 0)
}

func TestInventoryStoreReduceConcurrentRealDB(t *testing.T) {
	db, _ := mustOpenSchemaIntegrationDB(t)
	prepareInventorySchemaMigrations(t, db)
	store := newInventorys(&mysqlStore{db: db})
	goodsID := int32(time.Now().UnixNano() % 1_000_000_000)
	if err := store.Create(context.Background(), &do.InventoryDO{Goods: goodsID, Stocks: 25}); err != nil {
		t.Fatalf("Create(goodsID=%d) error = %v", goodsID, err)
	}

	const workers = 80
	var success atomic.Int32
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := store.Reduce(context.Background(), nil, uint64(goodsID), 1)
			switch {
			case err == nil:
				success.Add(1)
			case errors.IsCode(err, bizcode.ErrInvNotEnough):
			default:
				errs <- fmt.Errorf("Reduce(concurrent goodsID=%d) error = %v, want nil or ErrInvNotEnough", goodsID, err)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if got := success.Load(); got != 25 {
		t.Errorf("Reduce(concurrent goodsID=%d) successes = %d, want 25", goodsID, got)
	}
	assertInventoryAvailable(t, store, goodsID, 0, 25, 0)
}

func assertInventoryAvailable(t *testing.T, store *inventorys, goodsID, available, locked, sold int32) {
	t.Helper()
	inv, err := store.Get(context.Background(), uint64(goodsID))
	if err != nil {
		t.Fatalf("Get(goodsID=%d) error = %v", goodsID, err)
	}
	if inv.Available != available || inv.Locked != locked || inv.Sold != sold {
		t.Errorf("Get(goodsID=%d) = available:%d locked:%d sold:%d, want available:%d locked:%d sold:%d", goodsID, inv.Available, inv.Locked, inv.Sold, available, locked, sold)
	}
}
