package v1

import (
	"context"
	proto "goshop/api/goods/v1"

	"goshop/app/goods/srv/internal/domain/do"
)

type GoodsFilterRequest struct {
	*proto.GoodsFilterRequest
	CategoryIDs []interface{}
}

type GoodsStore interface {
	Create(ctx context.Context, goods *do.GoodsSearchDO) error
	Delete(ctx context.Context, ID uint64) error
	Update(ctx context.Context, goods *do.GoodsSearchDO) error
	Search(ctx context.Context, request *GoodsFilterRequest) (*do.GoodsSearchDOList, error)
}
