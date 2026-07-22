package db

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"goshop/app/pkg/options"
)

const schemaTestMySQLDSNEnv = "GOSHOP_SCHEMA_TEST_MYSQL_DSN"

func TestGoodsStartupValidationRealDB(t *testing.T) {
	db, dsn := mustOpenSchemaIntegrationDB(t)
	resetGoodsFactory()
	t.Cleanup(resetGoodsFactory)

	prepareCommerceSchemaMigrations(t, db)

	mysqlOpts := mustSchemaTestMySQLOptions(t, dsn)
	if _, err := GetDBFactoryOr(mysqlOpts); err != nil {
		t.Fatalf("GetDBFactoryOr(schema migrated db) error = %v", err)
	}
}

func mustOpenSchemaIntegrationDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv(schemaTestMySQLDSNEnv))
	if dsn == "" {
		t.Skipf("set %s to run real MySQL schema integration tests", schemaTestMySQLDSNEnv)
	}

	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("mysql.ParseDSN(%s) error = %v", schemaTestMySQLDSNEnv, err)
	}
	cfg.MultiStatements = true

	db, err := gorm.Open(gormmysql.Open(cfg.FormatDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open(schema test db) error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(schema test db) error = %v", err)
	}
	sqlDB.SetMaxOpenConns(32)
	sqlDB.SetMaxIdleConns(8)
	sqlDB.SetConnMaxLifetime(time.Minute)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db, dsn
}

func prepareCommerceSchemaMigrations(t *testing.T, db *gorm.DB) {
	t.Helper()

	dropStatements := []string{
		"DROP TABLE IF EXISTS `order_status_logs`",
		"DROP TABLE IF EXISTS `outbox_events`",
		"DROP TABLE IF EXISTS `shoppingcart`",
		"DROP TABLE IF EXISTS `ordergoods`",
		"DROP TABLE IF EXISTS `orderinfo`",
		"DROP TABLE IF EXISTS `goods`",
		"DROP TABLE IF EXISTS `goodscategorybrand`",
		"DROP TABLE IF EXISTS `banner`",
		"DROP TABLE IF EXISTS `brands`",
		"DROP TABLE IF EXISTS `category`",
	}
	for _, statement := range dropStatements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("cleanup schema test tables error = %v", err)
		}
	}
	t.Cleanup(func() {
		for _, statement := range dropStatements {
			_ = db.Exec(statement).Error
		}
	})

	for _, path := range commerceMigrationFiles(t) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		if err := db.Exec(string(content)).Error; err != nil {
			t.Fatalf("apply migration %s error = %v", filepath.Base(path), err)
		}
	}
}

func commerceMigrationFiles(t *testing.T) []string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../../../"))
	return []string{
		filepath.Join(root, "migrations/202607070001_goods_create_core_tables.up.sql"),
		filepath.Join(root, "migrations/202607070002_order_create_core_tables.up.sql"),
		filepath.Join(root, "migrations/202607080001_goods_add_outbox_events.up.sql"),
		filepath.Join(root, "migrations/202607100001_order_add_status_logs.up.sql"),
		filepath.Join(root, "migrations/202607220003_goods_order_add_money_fen_columns.up.sql"),
		filepath.Join(root, "migrations/202607220004_goods_order_drop_float_money_columns.up.sql"),
	}
}

func mustSchemaTestMySQLOptions(t *testing.T, dsn string) *options.MySQLOptions {
	t.Helper()

	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("mysql.ParseDSN(schema test dsn) error = %v", err)
	}
	host, port, err := splitMySQLAddr(cfg.Addr)
	if err != nil {
		t.Fatalf("splitMySQLAddr(%q) error = %v", cfg.Addr, err)
	}
	return &options.MySQLOptions{
		Host:                  host,
		Port:                  port,
		Username:              cfg.User,
		Password:              cfg.Passwd,
		Database:              cfg.DBName,
		MaxIdleConnections:    8,
		MaxOpenConnections:    32,
		MaxConnectionLifetime: time.Minute,
		LogLevel:              1,
	}
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

func resetGoodsFactory() {
	dbFactory = nil
	once = sync.Once{}
}
