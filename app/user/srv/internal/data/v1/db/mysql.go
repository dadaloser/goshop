package db

import (
	"fmt"
	"goshop/app/pkg/code"
	"goshop/app/pkg/options"
	errors2 "goshop/pkg/errors"
	"goshop/pkg/log"
	"sync"

	"gorm.io/gorm"

	"gorm.io/driver/mysql"
)

var (
	dbFactory *gorm.DB
	once      sync.Once
)

// 这个方法会返回gorm连接
// 还不够
// 这个方法应该返回的是全局的一个变量，如果一开始的时候没有初始化好，那么就初始化一次，后续呢直接拿到这个变量
func GetDBFactoryOr(mysqlOpts *options.MySQLOptions) (*gorm.DB, error) {
	if mysqlOpts == nil && dbFactory == nil {
		return nil, fmt.Errorf("failed to get mysql store fatory")
	}

	var err error
	once.Do(func() {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			mysqlOpts.Username,
			mysqlOpts.Password,
			mysqlOpts.Host,
			mysqlOpts.Port,
			mysqlOpts.Database)
		log.Infof("connecting mysql: host=%s port=%s database=%s user=%s",
			mysqlOpts.Host, mysqlOpts.Port, mysqlOpts.Database, mysqlOpts.Username)
		dbFactory, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			return
		}

		sqlDB, _ := dbFactory.DB()

		sqlDB.SetMaxOpenConns(mysqlOpts.MaxOpenConnections)
		sqlDB.SetMaxIdleConns(mysqlOpts.MaxIdleConnections)
		sqlDB.SetConnMaxLifetime(mysqlOpts.MaxConnectionLifetime)
		log.Infof("mysql connected: host=%s port=%s database=%s", mysqlOpts.Host, mysqlOpts.Port, mysqlOpts.Database)
	})

	if dbFactory == nil || err != nil {
		return nil, errors2.WrapC(err, code.ErrConnectDB, "failed to get mysql store factory")
	}
	return dbFactory, nil
}
