package db

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"goshop/app/pkg/code"
	appgorm "goshop/app/pkg/gorm"
	"goshop/app/pkg/options"
	"goshop/app/review/srv/internal/domain"
	errors2 "goshop/pkg/errors"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dbFactory *gorm.DB
	once      sync.Once
)

func GetDBFactoryOr(mysqlOpts *options.MySQLOptions) (*gorm.DB, error) {
	if mysqlOpts == nil && dbFactory == nil {
		return nil, fmt.Errorf("failed to get mysql store factory")
	}

	var initErr error
	once.Do(func() {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			mysqlOpts.Username,
			mysqlOpts.Password,
			mysqlOpts.Host,
			mysqlOpts.Port,
			mysqlOpts.Database)

		newLogger := logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.LogLevel(mysqlOpts.LogLevel),
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		)
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: newLogger,
		})
		if err != nil {
			initErr = err
			return
		}
		if err = db.Use(appgorm.NewResiliencePlugin(mysqlOpts.Resilience)); err != nil {
			initErr = err
			return
		}

		sqlDB, dbErr := db.DB()
		if dbErr != nil {
			initErr = fmt.Errorf("open sql db: %w", dbErr)
			return
		}
		dbFactory = db

		sqlDB.SetMaxOpenConns(mysqlOpts.MaxOpenConnections)
		sqlDB.SetMaxIdleConns(mysqlOpts.MaxIdleConnections)
		sqlDB.SetConnMaxLifetime(mysqlOpts.MaxConnectionLifetime)
		if mysqlOpts.AutoMigrate {
			if err = migrateReviewSchema(db); err != nil {
				_ = sqlDB.Close()
				dbFactory = nil
				initErr = err
				return
			}
		}
		if err = validateReviewSchema(db); err != nil {
			_ = sqlDB.Close()
			dbFactory = nil
			initErr = err
			return
		}
	})

	if dbFactory == nil || initErr != nil {
		return nil, errors2.WrapC(initErr, code.ErrConnectDB, "failed to get mysql store factory")
	}
	return dbFactory, nil
}

func migrateReviewSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("review schema migration failed: nil db")
	}
	if err := db.AutoMigrate(
		&domain.Review{},
		&domain.ReviewAppend{},
		&domain.ReviewReply{},
		&domain.Audit{},
		&domain.OutboxEvent{},
		&domain.Rating{},
	); err != nil {
		return fmt.Errorf("review schema migration failed: %w", err)
	}
	return nil
}

type schemaTableCheck struct {
	model    interface{ TableName() string }
	required []string
}

func validateReviewSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("review schema validation failed: nil db")
	}

	for _, table := range reviewSchemaChecks() {
		if !db.Migrator().HasTable(table.model) {
			return fmt.Errorf("review schema validation failed: required table %q does not exist", table.model.TableName())
		}
		for _, column := range table.required {
			if !db.Migrator().HasColumn(table.model, column) {
				return fmt.Errorf("review schema validation failed: required column %q.%q does not exist", table.model.TableName(), column)
			}
		}
	}

	return nil
}

func reviewSchemaChecks() []schemaTableCheck {
	return []schemaTableCheck{
		{
			model:    &domain.Review{},
			required: []string{"id", "user_id", "order_sn", "goods_id", "rating", "content", "status", "created_at", "updated_at"},
		},
		{
			model:    &domain.ReviewAppend{},
			required: []string{"id", "review_id", "content", "created_at"},
		},
		{
			model:    &domain.ReviewReply{},
			required: []string{"id", "review_id", "actor_user_id", "content", "created_at"},
		},
		{
			model:    &domain.Audit{},
			required: []string{"id", "review_id", "actor_user_id", "action", "from_status", "to_status", "request_id", "reason", "created_at"},
		},
		{
			model:    &domain.OutboxEvent{},
			required: []string{"id", "event_key", "goods_id", "event_type", "status", "retry_count", "next_attempt_at", "last_error", "created_at", "completed_at"},
		},
		{
			model:    &domain.Rating{},
			required: []string{"goods_id", "approved_count", "rating_sum", "average_milli", "rebuilt_at"},
		},
	}
}

func resetReviewFactory() {
	dbFactory = nil
	once = sync.Once{}
}
