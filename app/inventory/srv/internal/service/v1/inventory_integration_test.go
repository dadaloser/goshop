package v1

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	invdb "goshop/app/inventory/srv/internal/data/v1/db"
	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/inventory/srv/internal/domain/dto"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const inventoryTestMySQLDSNEnv = "GOSHOP_INVENTORY_TEST_MYSQL_DSN"

var inventoryIntegrationGoodsSeed atomic.Int64

func TestInventorySellConcurrentRealDB(t *testing.T) {
	db, dsn := mustOpenInventoryIntegrationDB(t)
	prepareInventoryIntegrationSchema(t, db)

	srv := mustNewInventoryIntegrationService(t, dsn)
	goodsID := nextInventoryIntegrationGoodsID()
	orderPrefix := fmt.Sprintf("inventory-realdb-%d", time.Now().UnixNano())
	seedInventoryIntegrationFixture(t, db, goodsID, 100, orderPrefix)

	const workers = 1000
	var successCount atomic.Int32
	var notEnoughCount atomic.Int32

	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			orderSn := fmt.Sprintf("%s-%d", orderPrefix, i)
			err := srv.Sell(context.Background(), orderSn, []do.GoodsDetail{{
				Goods: goodsID,
				Num:   1,
			}})
			switch {
			case err == nil:
				successCount.Add(1)
			case errors.IsCode(err, code.ErrInvNotEnough):
				notEnoughCount.Add(1)
			default:
				errCh <- fmt.Errorf("Inventory.Sell(orderSn=%q, goodsID=%d) error = %v, want nil or ErrInvNotEnough", orderSn, goodsID, err)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}

	if got, want := successCount.Load(), int32(100); got != want {
		t.Fatalf("Inventory.Sell(concurrent goodsID=%d) success count = %d, want %d", goodsID, got, want)
	}
	if got, want := notEnoughCount.Load(), int32(workers)-successCount.Load(); got != want {
		t.Fatalf("Inventory.Sell(concurrent goodsID=%d) not enough count = %d, want %d", goodsID, got, want)
	}

	assertInventoryState(t, srv, goodsID, inventoryStateExpectation{
		Stocks:    0,
		Total:     100,
		Available: 0,
		Locked:    100,
		Sold:      0,
	})
	assertReservedSellDetailCount(t, db, orderPrefix+"-%", 100)
}

func TestInventorySellDuplicateOrderRealDB(t *testing.T) {
	db, dsn := mustOpenInventoryIntegrationDB(t)
	prepareInventoryIntegrationSchema(t, db)

	srv := mustNewInventoryIntegrationService(t, dsn)
	goodsID := nextInventoryIntegrationGoodsID()
	orderSn := fmt.Sprintf("inventory-realdb-duplicate-%d", time.Now().UnixNano())
	seedInventoryIntegrationFixture(t, db, goodsID, 5, orderSn)

	detail := []do.GoodsDetail{{
		Goods: goodsID,
		Num:   2,
	}}

	if err := srv.Sell(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Sell(first orderSn=%q, goodsID=%d) error = %v", orderSn, goodsID, err)
	}
	if err := srv.Sell(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Sell(duplicate orderSn=%q, goodsID=%d) error = %v, want nil", orderSn, goodsID, err)
	}

	assertInventoryState(t, srv, goodsID, inventoryStateExpectation{
		Stocks:    3,
		Total:     5,
		Available: 3,
		Locked:    2,
		Sold:      0,
	})
	assertStockSellDetailStatus(t, db, orderSn, stockSellStatusReserved)
	assertStockSellDetailCount(t, db, orderSn, 1)
}

func TestInventoryConfirmRealDB(t *testing.T) {
	db, dsn := mustOpenInventoryIntegrationDB(t)
	prepareInventoryIntegrationSchema(t, db)

	srv := mustNewInventoryIntegrationService(t, dsn)
	goodsID := nextInventoryIntegrationGoodsID()
	orderSn := fmt.Sprintf("inventory-realdb-confirm-%d", time.Now().UnixNano())
	seedInventoryIntegrationFixture(t, db, goodsID, 5, orderSn)

	detail := []do.GoodsDetail{{
		Goods: goodsID,
		Num:   2,
	}}

	if err := srv.Sell(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Sell(orderSn=%q, goodsID=%d) error = %v", orderSn, goodsID, err)
	}
	if err := srv.Confirm(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Confirm(orderSn=%q, goodsID=%d) error = %v", orderSn, goodsID, err)
	}

	assertInventoryState(t, srv, goodsID, inventoryStateExpectation{
		Stocks:    3,
		Total:     5,
		Available: 3,
		Locked:    0,
		Sold:      2,
	})
	assertStockSellDetailStatus(t, db, orderSn, stockSellStatusConfirmed)
}

func TestInventoryReleaseRealDB(t *testing.T) {
	db, dsn := mustOpenInventoryIntegrationDB(t)
	prepareInventoryIntegrationSchema(t, db)

	srv := mustNewInventoryIntegrationService(t, dsn)
	goodsID := nextInventoryIntegrationGoodsID()
	orderSn := fmt.Sprintf("inventory-realdb-release-%d", time.Now().UnixNano())
	seedInventoryIntegrationFixture(t, db, goodsID, 5, orderSn)

	detail := []do.GoodsDetail{{
		Goods: goodsID,
		Num:   2,
	}}

	if err := srv.Sell(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Sell(orderSn=%q, goodsID=%d) error = %v", orderSn, goodsID, err)
	}
	if err := srv.Release(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Release(orderSn=%q, goodsID=%d) error = %v", orderSn, goodsID, err)
	}

	assertInventoryState(t, srv, goodsID, inventoryStateExpectation{
		Stocks:    5,
		Total:     5,
		Available: 5,
		Locked:    0,
		Sold:      0,
	})
	assertStockSellDetailStatus(t, db, orderSn, stockSellStatusReleased)
}

func TestInventoryConfirmIsIdempotentRealDB(t *testing.T) {
	db, dsn := mustOpenInventoryIntegrationDB(t)
	prepareInventoryIntegrationSchema(t, db)

	srv := mustNewInventoryIntegrationService(t, dsn)
	goodsID := nextInventoryIntegrationGoodsID()
	orderSn := fmt.Sprintf("inventory-realdb-confirm-idempotent-%d", time.Now().UnixNano())
	seedInventoryIntegrationFixture(t, db, goodsID, 5, orderSn)

	detail := []do.GoodsDetail{{
		Goods: goodsID,
		Num:   2,
	}}

	if err := srv.Sell(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Sell(orderSn=%q, goodsID=%d) error = %v", orderSn, goodsID, err)
	}
	if err := srv.Confirm(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Confirm(first orderSn=%q, goodsID=%d) error = %v", orderSn, goodsID, err)
	}
	if err := srv.Confirm(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Confirm(retry orderSn=%q, goodsID=%d) error = %v, want nil", orderSn, goodsID, err)
	}

	assertInventoryState(t, srv, goodsID, inventoryStateExpectation{
		Stocks:    3,
		Total:     5,
		Available: 3,
		Locked:    0,
		Sold:      2,
	})
	assertStockSellDetailStatus(t, db, orderSn, stockSellStatusConfirmed)
	assertStockSellDetailCount(t, db, orderSn, 1)
}

func TestInventoryReleaseAfterConfirmDoesNotRestoreStockRealDB(t *testing.T) {
	db, dsn := mustOpenInventoryIntegrationDB(t)
	prepareInventoryIntegrationSchema(t, db)

	srv := mustNewInventoryIntegrationService(t, dsn)
	goodsID := nextInventoryIntegrationGoodsID()
	orderSn := fmt.Sprintf("inventory-realdb-release-after-confirm-%d", time.Now().UnixNano())
	seedInventoryIntegrationFixture(t, db, goodsID, 5, orderSn)

	detail := []do.GoodsDetail{{
		Goods: goodsID,
		Num:   2,
	}}

	if err := srv.Sell(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Sell(orderSn=%q, goodsID=%d) error = %v", orderSn, goodsID, err)
	}
	if err := srv.Confirm(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Confirm(orderSn=%q, goodsID=%d) error = %v", orderSn, goodsID, err)
	}
	if err := srv.Release(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Release(after confirm orderSn=%q, goodsID=%d) error = %v, want nil", orderSn, goodsID, err)
	}

	assertInventoryState(t, srv, goodsID, inventoryStateExpectation{
		Stocks:    3,
		Total:     5,
		Available: 3,
		Locked:    0,
		Sold:      2,
	})
	assertStockSellDetailStatus(t, db, orderSn, stockSellStatusConfirmed)
}

func TestInventoryReleaseBeforeSellBlocksLateSellRealDB(t *testing.T) {
	db, dsn := mustOpenInventoryIntegrationDB(t)
	prepareInventoryIntegrationSchema(t, db)

	srv := mustNewInventoryIntegrationService(t, dsn)
	goodsID := nextInventoryIntegrationGoodsID()
	orderSn := fmt.Sprintf("inventory-realdb-release-before-sell-%d", time.Now().UnixNano())
	seedInventoryIntegrationFixture(t, db, goodsID, 5, orderSn)

	detail := []do.GoodsDetail{{
		Goods: goodsID,
		Num:   2,
	}}

	if err := srv.Release(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Release(before sell orderSn=%q, goodsID=%d) error = %v", orderSn, goodsID, err)
	}
	if err := srv.Sell(context.Background(), orderSn, detail); err != nil {
		t.Fatalf("Inventory.Sell(after release marker orderSn=%q, goodsID=%d) error = %v, want nil", orderSn, goodsID, err)
	}

	assertInventoryState(t, srv, goodsID, inventoryStateExpectation{
		Stocks:    5,
		Total:     5,
		Available: 5,
		Locked:    0,
		Sold:      0,
	})
	assertStockSellDetailStatus(t, db, orderSn, stockSellStatusReleased)
	assertStockSellDetailCount(t, db, orderSn, 1)
}

func TestInventoryAdjustmentAuditRealDB(t *testing.T) {
	db, dsn := mustOpenInventoryIntegrationDB(t)
	prepareInventoryIntegrationSchema(t, db)
	srv := mustNewInventoryIntegrationService(t, dsn)
	goodsID := nextInventoryIntegrationGoodsID()
	correlationID := fmt.Sprintf("adjust-%d", time.Now().UnixNano())
	seedInventoryIntegrationFixture(t, db, goodsID, 5, correlationID)
	t.Cleanup(func() { _ = db.Where("correlation_id = ?", correlationID).Delete(&do.InventoryAdjustmentDO{}).Error })

	err := srv.Adjust(context.Background(), &dto.InventoryDTO{InventoryDO: do.InventoryDO{Goods: goodsID, Stocks: 8}}, &do.InventoryAdjustmentDO{ActorUserID: 42, CorrelationID: correlationID, RequestID: "request-1", Reason: "cycle count correction"})
	if err != nil {
		t.Fatalf("Inventory.Adjust() error = %v", err)
	}
	items, total, err := srv.ListAdjustments(context.Background(), uint64(goodsID), 1, 20)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("ListAdjustments() total=%d items=%+v error=%v", total, items, err)
	}
	got := items[0]
	if got.BeforeAvailable != 5 || got.AfterAvailable != 8 || got.ActorUserID != 42 || got.CorrelationID != correlationID {
		t.Fatalf("adjustment audit = %+v", got)
	}
}

func mustOpenInventoryIntegrationDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()

	dsn := inventoryIntegrationDSNFromEnv()
	if dsn == "" {
		t.Skipf("set %s or INVENTORY_MYSQL_USERNAME/INVENTORY_MYSQL_PASSWORD[/HOST/PORT/DATABASE] to run real MySQL inventory integration tests", inventoryTestMySQLDSNEnv)
	}
	silenceInventoryIntegrationLogs()

	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open(real inventory test db) error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(real inventory test db) error = %v", err)
	}
	sqlDB.SetMaxOpenConns(128)
	sqlDB.SetMaxIdleConns(32)
	sqlDB.SetConnMaxLifetime(time.Minute)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db, dsn
}

func mustNewInventoryIntegrationService(t *testing.T, dsn string) *inventoryService {
	t.Helper()

	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("mysql.ParseDSN(real inventory test dsn) error = %v", err)
	}
	host, port, err := splitMySQLAddr(cfg.Addr)
	if err != nil {
		t.Fatalf("splitMySQLAddr(%q) error = %v", cfg.Addr, err)
	}

	mysqlOpts := &options.MySQLOptions{
		Host:                  host,
		Port:                  port,
		Username:              cfg.User,
		Password:              cfg.Passwd,
		Database:              cfg.DBName,
		MaxIdleConnections:    32,
		MaxOpenConnections:    128,
		MaxConnectionLifetime: time.Minute,
		LogLevel:              1,
	}

	dataFactory, err := invdb.GetDBFactoryOr(mysqlOpts)
	if err != nil {
		t.Fatalf("GetDBFactoryOr(real inventory test db) error = %v", err)
	}

	return &inventoryService{
		data: dataFactory,
		pool: nil,
	}
}

func prepareInventoryIntegrationSchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS inventory (
			id int NOT NULL AUTO_INCREMENT,
			add_time datetime(3) NULL,
			update_time datetime(3) NULL,
			deleted_at datetime(3) NULL,
			is_deleted tinyint(1) DEFAULT 0,
			goods int DEFAULT 0,
			stocks int DEFAULT 0,
			total int DEFAULT 0,
			available int DEFAULT 0,
			locked int DEFAULT 0,
			sold int DEFAULT 0,
			version int DEFAULT 0,
			PRIMARY KEY (id),
			KEY idx_inventory_goods (goods),
			KEY idx_inventory_deleted_at (deleted_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS stockselldetail (
			order_sn varchar(200) NOT NULL,
			status int NOT NULL,
			detail varchar(200) NOT NULL,
			UNIQUE KEY idx_order_sn (order_sn)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS inventory_adjustment_logs (
			id bigint unsigned NOT NULL AUTO_INCREMENT,
			goods_id int NOT NULL,
			before_available int NOT NULL,
			after_available int NOT NULL,
			actor_user_id int NOT NULL,
			correlation_id varchar(64) NOT NULL,
			request_id varchar(128) NOT NULL,
			reason varchar(255) NOT NULL,
			created_at datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			PRIMARY KEY (id),
			UNIQUE KEY uk_inventory_adjustment_correlation (correlation_id),
			KEY idx_inventory_adjustment_goods (goods_id, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare inventory integration schema error = %v", err)
		}
	}
}

func seedInventoryIntegrationFixture(t *testing.T, db *gorm.DB, goodsID int32, stocks int32, orderPrefix string) {
	t.Helper()

	if err := db.WithContext(context.Background()).Create(&do.InventoryDO{
		Goods:     goodsID,
		Stocks:    stocks,
		Total:     stocks,
		Available: stocks,
		Locked:    0,
		Sold:      0,
		Version:   0,
	}).Error; err != nil {
		t.Fatalf("Create(inventory fixture goodsID=%d) error = %v", goodsID, err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		_ = db.WithContext(ctx).Unscoped().
			Where("order_sn = ? OR order_sn LIKE ?", orderPrefix, orderPrefix+"-%").
			Delete(&do.StockSellDetailDO{}).Error
		_ = db.WithContext(ctx).Unscoped().Where("goods = ?", goodsID).Delete(&do.InventoryDO{}).Error
	})
}

func nextInventoryIntegrationGoodsID() int32 {
	return int32(time.Now().Unix()%1_000_000_000) + int32(inventoryIntegrationGoodsSeed.Add(1))
}

func splitMySQLAddr(addr string) (string, string, error) {
	if addr == "" {
		return "127.0.0.1", "3306", nil
	}
	host, port, err := net.SplitHostPort(addr)
	if err == nil {
		return host, port, nil
	}
	if strings.Count(addr, ":") == 0 {
		return addr, "3306", nil
	}
	return "", "", err
}

func inventoryIntegrationDSNFromEnv() string {
	if dsn := strings.TrimSpace(os.Getenv(inventoryTestMySQLDSNEnv)); dsn != "" {
		return dsn
	}

	username := strings.TrimSpace(os.Getenv("INVENTORY_MYSQL_USERNAME"))
	password := strings.TrimSpace(os.Getenv("INVENTORY_MYSQL_PASSWORD"))
	if username == "" || password == "" {
		return ""
	}

	host := strings.TrimSpace(os.Getenv("INVENTORY_MYSQL_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	port := strings.TrimSpace(os.Getenv("INVENTORY_MYSQL_PORT"))
	if port == "" {
		port = "3306"
	}
	database := strings.TrimSpace(os.Getenv("INVENTORY_MYSQL_DATABASE"))
	if database == "" {
		database = "goshop_inventory_srv"
	}

	cfg := mysqldriver.Config{
		User:                 username,
		Passwd:               password,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(host, port),
		DBName:               database,
		ParseTime:            true,
		Loc:                  time.Local,
		AllowNativePasswords: true,
	}
	return cfg.FormatDSN()
}

type inventoryStateExpectation struct {
	Stocks    int32
	Total     int32
	Available int32
	Locked    int32
	Sold      int32
}

func assertInventoryState(t *testing.T, srv *inventoryService, goodsID int32, want inventoryStateExpectation) {
	t.Helper()

	inv, err := srv.data.Inventories().Get(context.Background(), uint64(goodsID))
	if err != nil {
		t.Fatalf("Inventories().Get(goodsID=%d) error = %v", goodsID, err)
	}
	if got := inv.Stocks; got != want.Stocks {
		t.Errorf("Inventories().Get(goodsID=%d).Stocks = %d, want %d", goodsID, got, want.Stocks)
	}
	if got := inv.Total; got != want.Total {
		t.Errorf("Inventories().Get(goodsID=%d).Total = %d, want %d", goodsID, got, want.Total)
	}
	if got := inv.Available; got != want.Available {
		t.Errorf("Inventories().Get(goodsID=%d).Available = %d, want %d", goodsID, got, want.Available)
	}
	if got := inv.Locked; got != want.Locked {
		t.Errorf("Inventories().Get(goodsID=%d).Locked = %d, want %d", goodsID, got, want.Locked)
	}
	if got := inv.Sold; got != want.Sold {
		t.Errorf("Inventories().Get(goodsID=%d).Sold = %d, want %d", goodsID, got, want.Sold)
	}
}

func assertStockSellDetailStatus(t *testing.T, db *gorm.DB, orderSn string, wantStatus int32) {
	t.Helper()

	var detail do.StockSellDetailDO
	if err := db.WithContext(context.Background()).Where("order_sn = ?", orderSn).First(&detail).Error; err != nil {
		t.Fatalf("Get stock sell detail(orderSn=%q) error = %v", orderSn, err)
	}
	if got := detail.Status; got != wantStatus {
		t.Errorf("Get stock sell detail(orderSn=%q).Status = %d, want %d", orderSn, got, wantStatus)
	}
}

func assertStockSellDetailCount(t *testing.T, db *gorm.DB, orderSn string, wantCount int64) {
	t.Helper()

	var got int64
	if err := db.WithContext(context.Background()).Model(&do.StockSellDetailDO{}).Where("order_sn = ?", orderSn).Count(&got).Error; err != nil {
		t.Fatalf("Count stock sell detail(orderSn=%q) error = %v", orderSn, err)
	}
	if got != wantCount {
		t.Errorf("Count stock sell detail(orderSn=%q) = %d, want %d", orderSn, got, wantCount)
	}
}

func assertReservedSellDetailCount(t *testing.T, db *gorm.DB, orderPattern string, wantCount int64) {
	t.Helper()

	var got int64
	err := db.WithContext(context.Background()).
		Model(&do.StockSellDetailDO{}).
		Where("order_sn LIKE ?", orderPattern).
		Where("status = ?", stockSellStatusReserved).
		Count(&got).Error
	if err != nil {
		t.Fatalf("Count reserved stock sell detail(orderPattern=%q) error = %v", orderPattern, err)
	}
	if got != wantCount {
		t.Errorf("Count reserved stock sell detail(orderPattern=%q) = %d, want %d", orderPattern, got, wantCount)
	}
}

var inventoryIntegrationLogOnce sync.Once

func silenceInventoryIntegrationLogs() {
	inventoryIntegrationLogOnce.Do(func() {
		opts := log.NewOptions()
		opts.Level = "fatal"
		opts.OutputPaths = []string{"stderr"}
		opts.ErrorOutputPaths = []string{"stderr"}
		log.Init(opts)
	})
}
