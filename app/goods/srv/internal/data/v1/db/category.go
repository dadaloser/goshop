package db

import (
	"context"
	stderrors "errors"
	"goshop/app/pkg/bizcode"
	"goshop/pkg/errors"

	v1 "goshop/app/goods/srv/internal/data/v1"
	"goshop/app/goods/srv/internal/domain/do"

	"gorm.io/gorm"
)

type categories struct {
	db *gorm.DB
}

func newCategorys(factory *mysqlFactory) *categories {
	categories := &categories{
		db: factory.db,
	}
	return categories
}

//func NewCategories(db *gorm.DB) *categories {
//	return &categories{
//		db: db,
//	}
//}

func (c *categories) Get(ctx context.Context, ID uint64) (*do.CategoryDO, error) {
	if ID == 0 {
		return nil, errors.NewCode(bizcode.ErrCategoryNotFound, "category not found")
	}

	category := &do.CategoryDO{}

	err := c.db.WithContext(ctx).Preload("SubCategory").Preload("SubCategory.SubCategory").First(category, ID).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NewCode(bizcode.ErrCategoryNotFound, err.Error())
		}
		return nil, wrapDatabaseError(err, "get category")
	}
	return category, nil
}

func (c *categories) ListAll(ctx context.Context, orderBy []string) (*do.CategoryDOList, error) {
	ret := &do.CategoryDOList{}
	countQuery := c.db.WithContext(ctx).Model(&do.CategoryDO{}).Where("level = ?", 1)
	if err := countQuery.Count(&ret.TotalCount).Error; err != nil {
		return nil, wrapDatabaseError(err, "count root categories")
	}

	query := c.db.WithContext(ctx).Model(&do.CategoryDO{}).Where("level = ?", 1)
	query = applyOrderBy(query, orderBy, categoryOrderColumns)

	if err := query.Preload("SubCategory.SubCategory").Find(&ret.Items).Error; err != nil {
		return nil, wrapDatabaseError(err, "list categories")
	}
	return ret, nil
}

func (c *categories) Create(ctx context.Context, category *do.CategoryDO) error {
	if category == nil {
		return errors.NewCode(bizcode.ErrGoodsInvalid, "category is required")
	}

	tx := c.db.WithContext(ctx).Create(category)
	if tx.Error != nil {
		return wrapDatabaseError(tx.Error, "create category")
	}
	return nil
}

func (c *categories) Update(ctx context.Context, category *do.CategoryDO) error {
	if category == nil || category.ID <= 0 {
		return errors.NewCode(bizcode.ErrCategoryNotFound, "category not found")
	}

	tx := c.db.WithContext(ctx).Model(&do.CategoryDO{}).
		Where("id = ?", category.ID).
		Updates(map[string]interface{}{
			"name":               category.Name,
			"parent_category_id": category.ParentCategoryID,
			"level":              category.Level,
			"is_tab":             category.IsTab,
		})
	if tx.Error != nil {
		return wrapDatabaseError(tx.Error, "update category")
	}
	if tx.RowsAffected == 0 {
		if _, err := c.Get(ctx, uint64(category.ID)); err != nil {
			return err
		}
	}
	return nil
}

func (c *categories) Delete(ctx context.Context, ID uint64) error {
	if ID == 0 {
		return errors.NewCode(bizcode.ErrCategoryNotFound, "category not found")
	}

	tx := c.db.WithContext(ctx).Where("id = ?", ID).Delete(&do.CategoryDO{})
	if tx.Error != nil {
		return wrapDatabaseError(tx.Error, "delete category")
	}
	if tx.RowsAffected == 0 {
		return errors.NewCode(bizcode.ErrCategoryNotFound, "category not found")
	}
	return nil
}

var _ v1.CategoryStore = &categories{}
