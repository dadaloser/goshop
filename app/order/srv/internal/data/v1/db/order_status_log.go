package db

import (
	"context"
	v1 "goshop/app/order/srv/internal/data/v1"
	"goshop/app/order/srv/internal/domain/do"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
	"strings"

	"gorm.io/gorm"
)

type orderStatusLogs struct {
	db *gorm.DB
}

func newOrderStatusLogs(factory *dataFactory) *orderStatusLogs {
	return &orderStatusLogs{db: factory.db}
}

func (o *orderStatusLogs) Create(ctx context.Context, txn *gorm.DB, entry *do.OrderStatusLogDO) error {
	if entry == nil || strings.TrimSpace(entry.OrderSn) == "" || strings.TrimSpace(entry.ToStatus) == "" {
		return errors.WithCode(code2.ErrValidation, "order status log is required")
	}

	db := o.db
	if txn != nil {
		db = txn
	}
	if err := db.WithContext(ctx).Create(entry).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

var _ v1.OrderStatusLogStore = &orderStatusLogs{}
