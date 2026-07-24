package mysql

import (
	"fmt"
	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/pkg/code"
	appgorm "goshop/app/pkg/gorm"
	"goshop/app/pkg/options"
	"log"
	"os"
	"sync"
	"time"

	v12 "goshop/app/inventory/srv/internal/data/v1"
	"goshop/pkg/errors"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mysqlStore struct {
	db *gorm.DB
}

func (m *mysqlStore) Inventories() v12.InventoryStore {
	return newInventorys(m)
}

var _ v12.DataFactory = &mysqlStore{}

var (
	dbFactory v12.DataFactory
	once      sync.Once
)

// 对于复杂的初始化过程，使用工厂模式
func GetDBFactoryOr(mysqlOpts *options.MySQLOptions) (v12.DataFactory, error) {
	if mysqlOpts == nil && dbFactory == nil {
		return nil, fmt.Errorf("failed to get mysql store fatory")
	}

	var initErr error
	once.Do(func() {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			mysqlOpts.Username,
			mysqlOpts.Password,
			mysqlOpts.Host,
			mysqlOpts.Port,
			mysqlOpts.Database)

		//希望大家自己可以去封装logger
		newLogger := logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer（日志输出的目标，前缀和日志包含的内容——译者注）
			logger.Config{
				SlowThreshold:             time.Second,                         // 慢 SQL 阈值
				LogLevel:                  logger.LogLevel(mysqlOpts.LogLevel), // 日志级别
				IgnoreRecordNotFoundError: true,                                // 忽略ErrRecordNotFound（记录未找到）错误
				Colorful:                  false,                               // 禁用彩色打印
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
		dbFactory = &mysqlStore{
			db: db,
		}

		sqlDB.SetMaxOpenConns(mysqlOpts.MaxOpenConnections)
		sqlDB.SetMaxIdleConns(mysqlOpts.MaxIdleConnections)
		sqlDB.SetConnMaxLifetime(mysqlOpts.MaxConnectionLifetime)
		if err = validateInventorySchema(db); err != nil {
			_ = sqlDB.Close()
			dbFactory = nil
			initErr = err
			return
		}
	})

	if dbFactory == nil || initErr != nil {
		return nil, errors.WrapC(initErr, code.ErrConnectDB, "failed to get mysql store factory")
	}
	return dbFactory, nil
}

func (ds *mysqlStore) Begin() *gorm.DB {
	return ds.db.Begin()
}

type schemaTableCheck struct {
	model    interface{ TableName() string }
	required []string
}

func validateInventorySchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("inventory schema validation failed: nil db")
	}

	for _, table := range inventorySchemaChecks() {
		if !db.Migrator().HasTable(table.model) {
			return fmt.Errorf("inventory schema validation failed: required table %q does not exist", table.model.TableName())
		}
		for _, column := range table.required {
			if !db.Migrator().HasColumn(table.model, column) {
				return fmt.Errorf("inventory schema validation failed: required column %q.%q does not exist", table.model.TableName(), column)
			}
		}
	}

	return nil
}

func inventorySchemaChecks() []schemaTableCheck {
	return []schemaTableCheck{
		{
			model: &do.InventoryDO{},
			required: []string{
				"id",
				"add_time",
				"update_time",
				"deleted_at",
				"is_deleted",
				"goods",
				"stocks",
				"total",
				"available",
				"locked",
				"sold",
				"version",
			},
		},
		{
			model:    &do.StockSellDetailDO{},
			required: []string{"order_sn", "status", "detail"},
		},
		{
			model:    &do.InventoryAdjustmentDO{},
			required: []string{"id", "goods_id", "before_available", "after_available", "actor_user_id", "correlation_id", "request_id", "reason", "created_at"},
		},
	}
}
