package db

import (
	"fmt"
	"goshop/app/goods/srv/internal/data/v1"
	dv1 "goshop/app/goods/srv/internal/domain/do"
	"goshop/app/pkg/code"
	appgorm "goshop/app/pkg/gorm"
	"goshop/app/pkg/options"
	errors2 "goshop/pkg/errors"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
)

var (
	dbFactory v1.DataFactory
	once      sync.Once
)

// 工厂实现
type mysqlFactory struct {
	db *gorm.DB
}

func (mf *mysqlFactory) Begin() *gorm.DB {
	return mf.db.Begin()
}

func (mf *mysqlFactory) Goods() v1.GoodsStore {
	return newGoods(mf)
}

func (mf *mysqlFactory) Outbox() v1.OutboxStore {
	return newOutbox(mf)
}

func (mf *mysqlFactory) Categories() v1.CategoryStore {
	return newCategorys(mf)
}

func (mf *mysqlFactory) Brands() v1.BrandsStore {
	return newBrands(mf)
}

func (mf *mysqlFactory) Banners() v1.BannerStore {
	return newBanner(mf)
}

func (m *mysqlFactory) CategoryBrands() v1.GoodsCategoryBrandStore {
	return newCategoryBrands(m)
}

var _ v1.DataFactory = &mysqlFactory{}

// 这个方法会返回gorm连接
// 还不够
// 这个方法应该返回的是全局的一个变量，如果一开始的时候没有初始化好，那么就初始化一次，后续呢直接拿到这个变量
func GetDBFactoryOr(mysqlOpts *options.MySQLOptions) (v1.DataFactory, error) {
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
		dbFactory = &mysqlFactory{
			db: db,
		}

		sqlDB.SetMaxOpenConns(mysqlOpts.MaxOpenConnections)
		sqlDB.SetMaxIdleConns(mysqlOpts.MaxIdleConnections)
		sqlDB.SetConnMaxLifetime(mysqlOpts.MaxConnectionLifetime)
		if err = validateGoodsSchema(db); err != nil {
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

type schemaTableCheck struct {
	model     interface{ TableName() string }
	required  []string
	forbidden []string
}

func validateGoodsSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("goods schema validation failed: nil db")
	}

	for _, table := range goodsSchemaChecks() {
		if !db.Migrator().HasTable(table.model) {
			return fmt.Errorf("goods schema validation failed: required table %q does not exist", table.model.TableName())
		}
		for _, column := range table.required {
			if !db.Migrator().HasColumn(table.model, column) {
				return fmt.Errorf("goods schema validation failed: required column %q.%q does not exist", table.model.TableName(), column)
			}
		}
		for _, column := range table.forbidden {
			if db.Migrator().HasColumn(table.model, column) {
				return fmt.Errorf("goods schema validation failed: deprecated column %q.%q still exists", table.model.TableName(), column)
			}
		}
	}

	return nil
}

func goodsSchemaChecks() []schemaTableCheck {
	return []schemaTableCheck{
		{
			model: &dv1.GoodsDO{},
			required: []string{
				"id", "add_time", "update_time", "deleted_at", "is_deleted",
				"category_id", "brands_id", "on_sale", "ship_free", "is_new", "is_hot",
				"name", "goods_sn", "click_num", "sold_num", "fav_num",
				"market_price_fen", "shop_price_fen", "goods_brief", "goods_desc",
				"images", "desc_images", "goods_front_image",
			},
			forbidden: []string{"market_price", "shop_price"},
		},
		{
			model:    &dv1.CategoryDO{},
			required: []string{"id", "name", "parent_category_id", "level", "is_tab"},
		},
		{
			model:    &dv1.BrandsDO{},
			required: []string{"id", "name", "logo"},
		},
		{
			model:    &dv1.BannerDO{},
			required: []string{"id", "image", "url", "index"},
		},
		{
			model:    &dv1.GoodsCategoryBrandDO{},
			required: []string{"id", "category_id", "brands_id"},
		},
		{
			model: &dv1.OutboxEventDO{},
			required: []string{
				"id", "topic", "aggregate_type", "aggregate_id", "action",
				"payload", "status", "retry_count", "max_retry_count",
				"last_error", "next_attempt_at", "processing_lock",
			},
		},
	}
}
