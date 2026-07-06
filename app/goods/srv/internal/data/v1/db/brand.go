package db

import (
	"context"

	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	v1 "goshop/app/goods/srv/internal/data/v1"
	"goshop/app/goods/srv/internal/domain/do"
	metav1 "goshop/pkg/common/meta/v1"

	"gorm.io/gorm"
)

type brands struct {
	db *gorm.DB
}

func newBrands(factory *mysqlFactory) *brands {
	brands := &brands{
		db: factory.db,
	}
	return brands
}

//func NewBrands(db *gorm.DB) *brands {
//	return &brands{
//		db: db,
//	}
//}

func (b *brands) Get(ctx context.Context, ID uint64) (*do.BrandsDO, error) {
	brand := &do.BrandsDO{}
	err := b.db.WithContext(ctx).First(brand, ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrGoodsNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return brand, nil
}

func (b *brands) List(ctx context.Context, opts metav1.ListMeta, orderBy []string) (*do.BrandsDOList, error) {
	ret := &do.BrandsDOList{}
	query := b.db.WithContext(ctx).Model(&do.BrandsDO{})
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

func (b *brands) Create(ctx context.Context, txn *gorm.DB, brands *do.BrandsDO) error {
	db := b.db
	if txn != nil {
		db = txn
	}
	if err := db.WithContext(ctx).Create(brands).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

func (b *brands) Update(ctx context.Context, txn *gorm.DB, brands *do.BrandsDO) error {
	db := b.db
	if txn != nil {
		db = txn
	}
	tx := db.WithContext(ctx).Save(brands)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

func (b *brands) Delete(ctx context.Context, ID uint64) error {
	if err := b.db.WithContext(ctx).Where("id = ?", ID).Delete(&do.BrandsDO{}).Error; err != nil {
		return errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return nil
}

var _ v1.BrandsStore = &brands{}
