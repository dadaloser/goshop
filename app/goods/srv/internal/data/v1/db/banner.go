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

type banners struct {
	db *gorm.DB
}

func newBanner(factory *mysqlFactory) *banners {
	banners := &banners{
		db: factory.db,
	}
	return banners
}

func (b *banners) List(ctx context.Context, opts metav1.ListMeta, orderBy []string) (*do.BannerList, error) {
	ret := &do.BannerList{}
	query := b.db.WithContext(ctx).Model(&do.BannerDO{})
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

func (b *banners) Create(ctx context.Context, txn *gorm.DB, banner *do.BannerDO) error {
	db := b.db
	if txn != nil {
		db = txn
	}
	if err := db.WithContext(ctx).Create(banner).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func (b *banners) Update(ctx context.Context, txn *gorm.DB, banner *do.BannerDO) error {
	db := b.db
	if txn != nil {
		db = txn
	}
	if err := db.WithContext(ctx).Save(banner).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func (b *banners) Delete(ctx context.Context, ID uint64) error {
	if err := b.db.WithContext(ctx).Where("id = ?", ID).Delete(&do.BannerDO{}).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

var _ v1.BannerStore = &banners{}
