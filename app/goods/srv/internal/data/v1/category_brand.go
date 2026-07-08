package v1

import (
	"context"
	"goshop/app/goods/srv/internal/domain/do"

	metav1 "goshop/pkg/common/meta/v1"

	"gorm.io/gorm"
)

type GoodsCategoryBrandStore interface {
	List(ctx context.Context, opts metav1.ListMeta, orderBy []string) (*do.GoodsCategoryBrandList, error)
	ListByCategory(ctx context.Context, categoryID uint64, orderBy []string) (*do.GoodsCategoryBrandList, error)
	Create(ctx context.Context, txn *gorm.DB, gcb *do.GoodsCategoryBrandDO) error
	Update(ctx context.Context, txn *gorm.DB, gcb *do.GoodsCategoryBrandDO) error
	Delete(ctx context.Context, ID uint64) error
}
