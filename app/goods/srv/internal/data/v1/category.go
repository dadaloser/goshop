package v1

import (
	"context"
	"goshop/app/goods/srv/internal/domain/do"
)

type CategoryStore interface {
	Get(ctx context.Context, ID uint64) (*do.CategoryDO, error)
	ListAll(ctx context.Context, orderBy []string) (*do.CategoryDOList, error)
	Create(ctx context.Context, goods *do.CategoryDO) error
	Update(ctx context.Context, goods *do.CategoryDO) error
	Delete(ctx context.Context, ID uint64) error
}
