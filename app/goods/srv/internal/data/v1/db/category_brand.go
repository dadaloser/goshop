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

type categoryBrands struct {
	db *gorm.DB
}

func newCategoryBrands(factory *mysqlFactory) *categoryBrands {
	return &categoryBrands{db: factory.db}
}

func (cb *categoryBrands) List(ctx context.Context, opts metav1.ListMeta, orderBy []string) (*do.GoodsCategoryBrandList, error) {
	ret := &do.GoodsCategoryBrandList{}
	if opts.PageSize <= 0 {
		opts.PageSize = 10
	}

	countQuery := cb.db.WithContext(ctx).Model(&do.GoodsCategoryBrandDO{})
	if err := countQuery.Count(&ret.TotalCount).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

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
	tx := query.Limit(opts.PageSize).Find(&ret.Items)
	if tx.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return ret, nil
}

func (cb *categoryBrands) ListByCategory(ctx context.Context, categoryID uint64, orderBy []string) (*do.GoodsCategoryBrandList, error) {
	ret := &do.GoodsCategoryBrandList{}
	if categoryID == 0 {
		return ret, nil
	}

	countQuery := cb.db.WithContext(ctx).Model(&do.GoodsCategoryBrandDO{}).Where("category_id = ?", categoryID)
	if err := countQuery.Count(&ret.TotalCount).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	query := cb.db.WithContext(ctx).
		Where("category_id = ?", categoryID).
		Preload("Category").
		Preload("Brands")
	for _, value := range orderBy {
		query = query.Order(value)
	}
	tx := query.Find(&ret.Items)
	if tx.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return ret, nil
}

func (cb *categoryBrands) Create(ctx context.Context, txn *gorm.DB, gcb *do.GoodsCategoryBrandDO) error {
	if gcb == nil {
		return errors.WithCode(code.ErrGoodsInvalid, "category brand relation is required")
	}

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
	if gcb == nil || gcb.ID <= 0 {
		return errors.WithCode(code.ErrCategoryBrandNotFound, "category brand relation not found")
	}

	db := cb.db
	if txn != nil {
		db = txn
	}
	tx := db.WithContext(ctx).Model(&do.GoodsCategoryBrandDO{}).
		Where("id = ?", gcb.ID).
		Updates(map[string]interface{}{
			"category_id": gcb.CategoryID,
			"brands_id":   gcb.BrandsID,
		})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		exists, err := cb.exists(ctx, uint64(gcb.ID))
		if err != nil {
			return err
		}
		if !exists {
			return errors.WithCode(code.ErrCategoryBrandNotFound, "category brand relation not found")
		}
	}
	return nil
}

func (cb *categoryBrands) Delete(ctx context.Context, ID uint64) error {
	if ID == 0 {
		return errors.WithCode(code.ErrCategoryBrandNotFound, "category brand relation not found")
	}

	tx := cb.db.WithContext(ctx).Where("id = ?", ID).Delete(&do.GoodsCategoryBrandDO{})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrCategoryBrandNotFound, "category brand relation not found")
	}
	return nil
}

var _ v1.GoodsCategoryBrandStore = &categoryBrands{}

func (cb *categoryBrands) exists(ctx context.Context, id uint64) (bool, error) {
	var relation do.GoodsCategoryBrandDO
	err := cb.db.WithContext(ctx).Select("id").First(&relation, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return true, nil
}
