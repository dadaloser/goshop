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

func (o *orderStatusLogs) ListByOrderSn(ctx context.Context, orderSn string) ([]*do.OrderStatusLogDO, error) {
	orderSn = strings.TrimSpace(orderSn)
	if orderSn == "" {
		return nil, errors.WithCode(code2.ErrValidation, "order_sn is required")
	}

	var entries []*do.OrderStatusLogDO
	if err := o.db.WithContext(ctx).
		Where("order_sn = ?", orderSn).
		Order("add_time asc, id asc").
		Find(&entries).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return entries, nil
}

var _ v1.OrderStatusLogStore = &orderStatusLogs{}
