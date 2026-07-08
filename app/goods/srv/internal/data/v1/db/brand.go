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
	if ID == 0 {
		return nil, errors.WithCode(code.ErrBrandNotFound, "brand not found")
	}

	brand := &do.BrandsDO{}
	err := b.db.WithContext(ctx).First(brand, ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrBrandNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return brand, nil
}

func (b *brands) List(ctx context.Context, opts metav1.ListMeta, orderBy []string) (*do.BrandsDOList, error) {
	ret := &do.BrandsDOList{}
	if opts.PageSize <= 0 {
		opts.PageSize = 10
	}

	countQuery := b.db.WithContext(ctx).Model(&do.BrandsDO{})
	if err := countQuery.Count(&ret.TotalCount).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

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
	tx := query.Limit(opts.PageSize).Find(&ret.Items)
	if tx.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return ret, nil
}

func (b *brands) Create(ctx context.Context, txn *gorm.DB, brands *do.BrandsDO) error {
	if brands == nil {
		return errors.WithCode(code.ErrGoodsInvalid, "brand is required")
	}

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
	if brands == nil || brands.ID <= 0 {
		return errors.WithCode(code.ErrBrandNotFound, "brand not found")
	}

	db := b.db
	if txn != nil {
		db = txn
	}
	tx := db.WithContext(ctx).Model(&do.BrandsDO{}).
		Where("id = ?", brands.ID).
		Updates(map[string]interface{}{
			"name": brands.Name,
			"logo": brands.Logo,
		})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		if _, err := b.Get(ctx, uint64(brands.ID)); err != nil {
			return err
		}
	}
	return nil
}

func (b *brands) Delete(ctx context.Context, ID uint64) error {
	if ID == 0 {
		return errors.WithCode(code.ErrBrandNotFound, "brand not found")
	}

	tx := b.db.WithContext(ctx).Where("id = ?", ID).Delete(&do.BrandsDO{})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrBrandNotFound, "brand not found")
	}
	return nil
}

var _ v1.BrandsStore = &brands{}
