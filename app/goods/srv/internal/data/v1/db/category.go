package db

import (
	"context"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
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
	category := &do.CategoryDO{}

	err := c.db.Preload("SubCategory").Preload("SubCategory.SubCategory").First(category, ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrCategoryNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return category, nil
}

func (c *categories) ListAll(ctx context.Context, orderBy []string) (*do.CategoryDOList, error) {
	ret := &do.CategoryDOList{}
	query := c.db

	for _, value := range orderBy {
		query = query.Order(value)
	}

	d := query.Where("level=1").Preload("SubCategory.SubCategory").Find(&ret.Items)
	return ret, d.Error
}

func (c *categories) Create(ctx context.Context, category *do.CategoryDO) error {
	tx := c.db.Create(category)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

func (c *categories) Update(ctx context.Context, category *do.CategoryDO) error {
	tx := c.db.Save(category)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

func (c *categories) Delete(ctx context.Context, ID uint64) error {
	return c.db.Where("id = ?", ID).Delete(&do.GoodsDO{}).Error
}

var _ v1.CategoryStore = &categories{}
