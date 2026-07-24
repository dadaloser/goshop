package data

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"goshop/app/review/srv/internal/domain"

	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const reviewTestMySQLDSNEnv = "GOSHOP_REVIEW_TEST_MYSQL_DSN"

func TestReviewRatingOutboxSurvivesRestartRealDB(t *testing.T) {
	db, _ := mustOpenReviewIntegrationDB(t)
	prepareReviewIntegrationSchema(t, db)

	store := NewStore(db)
	ctx := context.Background()
	review := &domain.Review{
		UserID:  11,
		OrderSN: "review-realdb-order-1",
		GoodsID: 101,
		Rating:  5,
		Content: "great",
	}
	if err := store.Create(ctx, review); err != nil {
		t.Fatalf("Create(review) error = %v", err)
	}
	if err := store.Moderate(ctx, review.ID, domain.StatusApproved, 99, "req-1", "looks good"); err != nil {
		t.Fatalf("Moderate(reviewID=%d) error = %v", review.ID, err)
	}

	var event domain.OutboxEvent
	if err := db.WithContext(ctx).Where("goods_id = ?", review.GoodsID).First(&event).Error; err != nil {
		t.Fatalf("lookup outbox event error = %v", err)
	}
	if got := event.Status; got != "PENDING" {
		t.Fatalf("outbox status after moderate = %q, want %q", got, "PENDING")
	}

	restarted := NewStore(db)
	if err := restarted.ProcessOutbox(ctx, 10); err != nil {
		t.Fatalf("ProcessOutbox() error = %v", err)
	}

	rating, err := restarted.GetRating(ctx, review.GoodsID)
	if err != nil {
		t.Fatalf("GetRating(goodsID=%d) error = %v", review.GoodsID, err)
	}
	if rating.ApprovedCount != 1 || rating.RatingSum != 5 || rating.AverageMilli != 5000 {
		t.Fatalf("rating = %+v, want approved=1 sum=5 average=5000", rating)
	}

	if err := db.WithContext(ctx).First(&event, event.ID).Error; err != nil {
		t.Fatalf("reload outbox event error = %v", err)
	}
	if got := event.Status; got != "DONE" {
		t.Fatalf("outbox status after process = %q, want %q", got, "DONE")
	}
	if event.CompletedAt == nil {
		t.Fatal("outbox completed_at = nil, want timestamp")
	}
}

func mustOpenReviewIntegrationDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()

	dsn := reviewIntegrationDSNFromEnv()
	if dsn == "" {
		t.Skipf("set %s or REVIEW_MYSQL_USERNAME/REVIEW_MYSQL_PASSWORD[/HOST/PORT/DATABASE] to run real MySQL review integration tests", reviewTestMySQLDSNEnv)
	}

	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("mysql.ParseDSN(review test dsn) error = %v", err)
	}
	cfg.MultiStatements = true

	db, err := gorm.Open(gormmysql.Open(cfg.FormatDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open(review test db) error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(review test db) error = %v", err)
	}
	sqlDB.SetMaxOpenConns(32)
	sqlDB.SetMaxIdleConns(8)
	sqlDB.SetConnMaxLifetime(time.Minute)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db, dsn
}

func reviewIntegrationDSNFromEnv() string {
	if dsn := strings.TrimSpace(os.Getenv(reviewTestMySQLDSNEnv)); dsn != "" {
		return dsn
	}

	username := strings.TrimSpace(os.Getenv("REVIEW_MYSQL_USERNAME"))
	password := strings.TrimSpace(os.Getenv("REVIEW_MYSQL_PASSWORD"))
	if username == "" || password == "" {
		return ""
	}

	host := strings.TrimSpace(os.Getenv("REVIEW_MYSQL_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	port := strings.TrimSpace(os.Getenv("REVIEW_MYSQL_PORT"))
	if port == "" {
		port = "3306"
	}
	database := strings.TrimSpace(os.Getenv("REVIEW_MYSQL_DATABASE"))
	if database == "" {
		database = "goshop_review_srv"
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

func prepareReviewIntegrationSchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	dropStatements := []string{
		"DROP TABLE IF EXISTS `review_product_ratings`",
		"DROP TABLE IF EXISTS `review_outbox_events`",
		"DROP TABLE IF EXISTS `review_audit_logs`",
		"DROP TABLE IF EXISTS `review_replies`",
		"DROP TABLE IF EXISTS `review_appends`",
		"DROP TABLE IF EXISTS `reviews`",
	}
	for _, statement := range dropStatements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("cleanup review integration schema error = %v", err)
		}
	}
	t.Cleanup(func() {
		for _, statement := range dropStatements {
			_ = db.Exec(statement).Error
		}
	})

	for _, path := range reviewMigrationFiles(t) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		if err = db.Exec(string(content)).Error; err != nil {
			t.Fatalf("apply migration %s error = %v", filepath.Base(path), err)
		}
	}
}

func reviewMigrationFiles(t *testing.T) []string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	return []string{
		filepath.Join(root, "migrations/202607230005_review_domain.up.sql"),
	}
}
