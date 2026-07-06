package db

import (
	"context"

	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	v1 "goshop/app/goods/srv/internal/data/v1"
	"goshop/app/goods/srv/internal/domain/do"
	metav1 "goshop/pkg/common/meta/v1"

	"gorm.io/gorm"
)

type categoryBrands struct {
	db *gorm.DB
}

func newCategoryBrands(factory *mysqlFactory) *categoryBrands {
	return &categoryBrands{db: factory.db}
}

func (cb *categoryBrands) List(ctx context.Context, opts metav1.ListMeta, orderBy []string) (*do.GoodsCategoryBrandList, error) {
	ret := &do.GoodsCategoryBrandList{}
	query := cb.db.WithContext(ctx).Preload("Category").Preload("Brands")
	for _, value := range orderBy {
		query = query.Order(value)
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 10
	}
	if opts.Page > 0 {
		query = query.Offset((opts.Page - 1) * opts.PageSize)
	}
	tx := query.Limit(opts.PageSize).Find(&ret.Items).Count(&ret.TotalCount)
	if tx.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return ret, nil
}

func (cb *categoryBrands) Create(ctx context.Context, txn *gorm.DB, gcb *do.GoodsCategoryBrandDO) error {
	db := cb.db
	if txn != nil {
		db = txn
	}
	if err := db.WithContext(ctx).Create(gcb).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func (cb *categoryBrands) Update(ctx context.Context, txn *gorm.DB, gcb *do.GoodsCategoryBrandDO) error {
	db := cb.db
	if txn != nil {
		db = txn
	}
	if err := db.WithContext(ctx).Save(gcb).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func (cb *categoryBrands) Delete(ctx context.Context, ID uint64) error {
	if err := cb.db.WithContext(ctx).Where("id = ?", ID).Delete(&do.GoodsCategoryBrandDO{}).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

var _ v1.GoodsCategoryBrandStore = &categoryBrands{}
