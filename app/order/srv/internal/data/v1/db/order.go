package db

import (
	"context"
	"strings"

	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	v1 "goshop/app/order/srv/internal/data/v1"
	"goshop/app/order/srv/internal/domain/do"
	metav1 "goshop/pkg/common/meta/v1"

	"gorm.io/gorm"
)

type orders struct {
	db *gorm.DB
}

func newOrders(factory *dataFactory) *orders {
	return &orders{
		db: factory.db,
	}
}

func (o *orders) Get(ctx context.Context, orderSn string) (*do.OrderInfoDO, error) {
	var order do.OrderInfoDO

	//加链路
	err := o.db.WithContext(ctx).Preload("OrderGoods").Where("order_sn = ?", orderSn).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrOrderNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return &order, nil
}

func (o *orders) List(ctx context.Context, userID uint64, meta metav1.ListMeta, orderBy []string) (*do.OrderInfoDOList, error) {
	ret := &do.OrderInfoDOList{}
	if userID == 0 {
		return ret, nil
	}

	//分页
	var limit, offset int
	if meta.PageSize == 0 {
		limit = 10
	} else {
		limit = meta.PageSize
	}

	if meta.Page > 0 {
		offset = (meta.Page - 1) * limit
	}

	countQuery := o.db.WithContext(ctx).Model(&do.OrderInfoDO{})
	countQuery = countQuery.Where("user = ?", userID)
	if err := countQuery.Count(&ret.TotalCount).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	//排序
	query := o.db.WithContext(ctx).Model(&do.OrderInfoDO{}).Preload("OrderGoods")
	query = query.Where("user = ?", userID)
	for _, value := range orderBy {
		query = query.Order(value)
	}

	d := query.Offset(offset).Limit(limit).Find(&ret.Items)
	if d.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, d.Error.Error())
	}
	return ret, nil
}

// Create 创建订单之后要删除对应的购物车记录
func (o *orders) Create(ctx context.Context, txn *gorm.DB, order *do.OrderInfoDO) error {
	db := o.db
	if txn != nil {
		db = txn
	}
	return db.WithContext(ctx).Create(order).Error
}

func (o *orders) DeleteByOrderSn(ctx context.Context, txn *gorm.DB, orderSn string) error {
	orderSn = strings.TrimSpace(orderSn)
	if orderSn == "" {
		return nil
	}

	db := o.db
	if txn != nil {
		db = txn
	}

	var order do.OrderInfoDO
	err := db.WithContext(ctx).Where("order_sn = ?", orderSn).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}

	if err := db.WithContext(ctx).Where("`order` = ?", order.ID).Delete(&do.OrderGoods{}).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	if err := db.WithContext(ctx).Where("id = ?", order.ID).Delete(&do.OrderInfoDO{}).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func (o *orders) Update(ctx context.Context, txn *gorm.DB, order *do.OrderInfoDO) error {
	db := o.db
	if txn != nil {
		db = txn
	}
	query := db.WithContext(ctx).Model(&do.OrderInfoDO{})
	if order.ID > 0 {
		query = query.Where("id = ?", order.ID)
	} else {
		query = query.Where("order_sn = ?", order.OrderSn)
	}
	tx := query.Updates(map[string]interface{}{
		"status": order.Status,
	})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrOrderNotFound, "order not found")
	}
	return nil
}

var _ v1.OrderStore = &orders{}
