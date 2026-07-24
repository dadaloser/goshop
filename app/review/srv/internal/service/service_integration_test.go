package service

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"goshop/app/review/srv/internal/data"
	"goshop/app/review/srv/internal/domain"

	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const reviewServiceTestMySQLDSNEnv = "GOSHOP_REVIEW_TEST_MYSQL_DSN"

func TestReviewPurchaseModerationReplyAndAggregateRealDB(t *testing.T) {
	db := mustOpenReviewServiceIntegrationDB(t)
	prepareReviewServiceIntegrationSchema(t, db)

	ctx := context.Background()
	orderSN := "review-e2e-order-1"
	if err := db.WithContext(ctx).Exec(
		"INSERT INTO orderinfo (`user`, order_sn, status, order_mount_fen, address, signer_name, singer_mobile, post) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		11, orderSN, "TRADE_FINISHED", 1999, "Shanghai Pudong", "buyer", "13800138000", "fast delivery",
	).Error; err != nil {
		t.Fatalf("insert orderinfo error = %v", err)
	}
	var orderID int32
	if err := db.WithContext(ctx).Raw("SELECT id FROM orderinfo WHERE order_sn = ?", orderSN).Scan(&orderID).Error; err != nil {
		t.Fatalf("lookup order id error = %v", err)
	}
	if err := db.WithContext(ctx).Exec(
		"INSERT INTO ordergoods (`order`, goods, goods_name, goods_image, goods_price_fen, nums) VALUES (?, ?, ?, ?, ?, ?)",
		orderID, 101, "goods-101", "goods-101.png", 1999, 1,
	).Error; err != nil {
		t.Fatalf("insert ordergoods error = %v", err)
	}

	repo := data.NewStore(db)
	svc := New(repo, NewDBOrderVerifier(db))

	review, err := svc.Create(ctx, 11, orderSN, 101, 5, "great product")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if review.Status != domain.StatusPending {
		t.Fatalf("review status = %q, want %q", review.Status, domain.StatusPending)
	}

	approved, err := svc.Moderate(ctx, review.ID, domain.StatusApproved, 9001, "req-moderate-1", "looks good")
	if err != nil {
		t.Fatalf("Moderate() error = %v", err)
	}
	if approved.Status != domain.StatusApproved {
		t.Fatalf("approved status = %q, want %q", approved.Status, domain.StatusApproved)
	}

	replied, err := svc.Reply(ctx, review.ID, 7001, "thanks for your feedback", "req-reply-1")
	if err != nil {
		t.Fatalf("Reply() error = %v", err)
	}
	if replied.Reply.Content != "thanks for your feedback" || replied.Reply.ActorUserID != 7001 {
		t.Fatalf("reply = %+v", replied.Reply)
	}

	if err := repo.ProcessOutbox(ctx, 10); err != nil {
		t.Fatalf("ProcessOutbox() error = %v", err)
	}

	rating, err := svc.GetRating(ctx, 101)
	if err != nil {
		t.Fatalf("GetRating() error = %v", err)
	}
	if rating.ApprovedCount != 1 || rating.RatingSum != 5 || rating.AverageMilli != 5000 {
		t.Fatalf("rating = %+v, want approved=1 sum=5 average=5000", rating)
	}

	reviews, total, err := svc.List(ctx, 101, 0, domain.StatusApproved, 1, 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(reviews) != 1 || reviews[0].Reply.Content != "thanks for your feedback" {
		t.Fatalf("List() total=%d reviews=%+v", total, reviews)
	}

	var audits int64
	if err := db.WithContext(ctx).Model(&domain.Audit{}).Where("review_id = ?", review.ID).Count(&audits).Error; err != nil {
		t.Fatalf("count audits error = %v", err)
	}
	if audits != 2 {
		t.Fatalf("audit rows = %d, want 2", audits)
	}
}

func mustOpenReviewServiceIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := reviewServiceIntegrationDSNFromEnv()
	if dsn == "" {
		t.Skipf("set %s or REVIEW_MYSQL_USERNAME/REVIEW_MYSQL_PASSWORD[/HOST/PORT/DATABASE] to run real MySQL review service integration tests", reviewServiceTestMySQLDSNEnv)
	}

	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("mysql.ParseDSN(review service test dsn) error = %v", err)
	}
	cfg.MultiStatements = true

	db, err := gorm.Open(gormmysql.Open(cfg.FormatDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open(review service test db) error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(review service test db) error = %v", err)
	}
	sqlDB.SetMaxOpenConns(32)
	sqlDB.SetMaxIdleConns(8)
	sqlDB.SetConnMaxLifetime(time.Minute)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

func reviewServiceIntegrationDSNFromEnv() string {
	if dsn := strings.TrimSpace(os.Getenv(reviewServiceTestMySQLDSNEnv)); dsn != "" {
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

func prepareReviewServiceIntegrationSchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	dropStatements := []string{
		"DROP TABLE IF EXISTS `review_product_ratings`",
		"DROP TABLE IF EXISTS `review_outbox_events`",
		"DROP TABLE IF EXISTS `review_audit_logs`",
		"DROP TABLE IF EXISTS `review_replies`",
		"DROP TABLE IF EXISTS `review_appends`",
		"DROP TABLE IF EXISTS `reviews`",
		"DROP TABLE IF EXISTS `payment_reconciliation_items`",
		"DROP TABLE IF EXISTS `payment_reconciliation_runs`",
		"DROP TABLE IF EXISTS `payment_events`",
		"DROP TABLE IF EXISTS `order_refund_outbox`",
		"DROP TABLE IF EXISTS `order_refund_requests`",
		"DROP TABLE IF EXISTS `order_status_logs`",
		"DROP TABLE IF EXISTS `shoppingcart`",
		"DROP TABLE IF EXISTS `ordergoods`",
		"DROP TABLE IF EXISTS `orderinfo`",
	}
	for _, statement := range dropStatements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("cleanup review service integration schema error = %v", err)
		}
	}
	t.Cleanup(func() {
		for _, statement := range dropStatements {
			_ = db.Exec(statement).Error
		}
	})

	for _, path := range reviewServiceMigrationFiles(t) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", path, err)
		}
		if err = db.Exec(string(content)).Error; err != nil {
			t.Fatalf("apply migration %s error = %v", filepath.Base(path), err)
		}
	}
}

func reviewServiceMigrationFiles(t *testing.T) []string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../"))
	return []string{
		filepath.Join(root, "migrations/202607070002_order_create_core_tables.up.sql"),
		filepath.Join(root, "migrations/202607100001_order_add_status_logs.up.sql"),
		filepath.Join(root, "migrations/202607220004_order_add_money_fen_columns.up.sql"),
		filepath.Join(root, "migrations/202607220006_order_drop_float_money_columns.up.sql"),
		filepath.Join(root, "migrations/202607230003_order_add_payment_events.up.sql"),
		filepath.Join(root, "migrations/202607240001_payment_refund_outbox_reconciliation.up.sql"),
		filepath.Join(root, "migrations/202607230005_review_domain.up.sql"),
	}
}
