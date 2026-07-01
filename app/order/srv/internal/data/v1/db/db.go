package db

import (
	"errors"
	"fmt"
	v1 "goshop/app/order/srv/internal/data/v1"
	"goshop/app/pkg/code"
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

func GetDataFactoryOr(mysqlOpts *options.MySQLOptions) (v1.DataFactory, error) {
	if mysqlOpts == nil && data == nil {
		return nil, errors.New("failed to get data store factory")
	}
	var err error
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
			return
		}

		sqlDB, _ := db.DB()
		sqlDB.SetMaxOpenConns(mysqlOpts.MaxOpenConnections)
		sqlDB.SetMaxIdleConns(mysqlOpts.MaxIdleConnections)
		sqlDB.SetConnMaxLifetime(mysqlOpts.MaxConnectionLifetime)

		data = &dataFactory{
			db: db,
		}
	})

	if data == nil || err != nil {
		return nil, errors2.WrapC(err, code.ErrConnectDB, "failed to get data store factory")
	}
	return data, nil
}
