package v1

import (
	"context"
	"goshop/app/goods/srv/internal/domain/do"

	metav1 "goshop/pkg/common/meta/v1"

	"gorm.io/gorm"
)

type BrandsStore interface {
	List(ctx context.Context, opts metav1.ListMeta, orderBy []string) (*do.BrandsDOList, error)
	Create(ctx context.Context, txn *gorm.DB, brands *do.BrandsDO) error
	Update(ctx context.Context, txn *gorm.DB, brands *do.BrandsDO) error
	Delete(ctx context.Context, ID uint64) error
	Get(ctx context.Context, ID uint64) (*do.BrandsDO, error)
}
