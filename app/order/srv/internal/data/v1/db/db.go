package db

import (
	"errors"
	"fmt"
	v1 "goshop/app/order/srv/internal/data/v1"
	dv1 "goshop/app/order/srv/internal/domain/do"
	"goshop/app/pkg/code"
	appgorm "goshop/app/pkg/gorm"
	"goshop/app/pkg/options"
	errors2 "goshop/pkg/errors"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type dataFactory struct {
	db *gorm.DB
}

func (df *dataFactory) Orders() v1.OrderStore {
	return newOrders(df)
}

func (df *dataFactory) OrderStatusLogs() v1.OrderStatusLogStore {
	return newOrderStatusLogs(df)
}

func (df *dataFactory) ShopCarts() v1.ShopCartStore {
	return newShopCarts(df)
}

func (df *dataFactory) Begin() *gorm.DB {
	return df.db.Begin()
}

var _ v1.DataFactory = &dataFactory{}

var (
	data v1.DataFactory
	once sync.Once
)

// GormDB returns the validated order database for co-located transactional domains.
func GormDB() *gorm.DB {
	if factory, ok := data.(*dataFactory); ok {
		return factory.db
	}
	return nil
}

func GetDataFactoryOr(mysqlOpts *options.MySQLOptions) (v1.DataFactory, error) {
	if mysqlOpts == nil && data == nil {
		return nil, errors.New("failed to get data store factory")
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
		sqlDB.SetMaxOpenConns(mysqlOpts.MaxOpenConnections)
		sqlDB.SetMaxIdleConns(mysqlOpts.MaxIdleConnections)
		sqlDB.SetConnMaxLifetime(mysqlOpts.MaxConnectionLifetime)
		if err = validateOrderSchema(db); err != nil {
			_ = sqlDB.Close()
			initErr = err
			return
		}

		data = &dataFactory{
			db: db,
		}
	})

	if data == nil || initErr != nil {
		return nil, errors2.WrapC(initErr, code.ErrConnectDB, "failed to get data store factory")
	}
	return data, nil
}

type schemaTableCheck struct {
	model     interface{ TableName() string }
	required  []string
	forbidden []string
}

func validateOrderSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("order schema validation failed: nil db")
	}

	for _, table := range orderSchemaChecks() {
		if !db.Migrator().HasTable(table.model) {
			return fmt.Errorf("order schema validation failed: required table %q does not exist", table.model.TableName())
		}
		for _, column := range table.required {
			if !db.Migrator().HasColumn(table.model, column) {
				return fmt.Errorf("order schema validation failed: required column %q.%q does not exist", table.model.TableName(), column)
			}
		}
		for _, column := range table.forbidden {
			if db.Migrator().HasColumn(table.model, column) {
				return fmt.Errorf("order schema validation failed: deprecated column %q.%q still exists", table.model.TableName(), column)
			}
		}
	}

	return nil
}

func orderSchemaChecks() []schemaTableCheck {
	return []schemaTableCheck{
		{
			model: &dv1.OrderInfoDO{},
			required: []string{
				"id", "add_time", "update_time", "deleted_at", "is_deleted",
				"user", "order_sn", "pay_type", "status", "trade_no",
				"order_mount_fen", "pay_time", "address", "signer_name",
				"singer_mobile", "post",
			},
			forbidden: []string{"order_mount"},
		},
		{
			model: &dv1.OrderGoods{},
			required: []string{
				"id", "add_time", "update_time", "deleted_at", "is_deleted",
				"order", "goods", "goods_name", "goods_image", "goods_price_fen", "nums",
			},
			forbidden: []string{"goods_price"},
		},
		{
			model:    &dv1.ShoppingCartDO{},
			required: []string{"id", "add_time", "update_time", "deleted_at", "is_deleted", "user", "goods", "nums", "checked"},
		},
		{
			model: &dv1.OrderStatusLogDO{},
			required: []string{
				"id", "add_time", "update_time", "deleted_at", "is_deleted",
				"order_id", "order_sn", "from_status", "to_status", "reason", "source", "operator",
			},
		},
	}
}
