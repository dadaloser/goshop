package v1

import (
	"context"
	gpb "goshop/api/goods/v1"
	"goshop/app/goshop/api/internal/data"
)

//服务接口要与app/goods/srv/internal/service/v1/goods.go的GoodsSrv接口保持一致

type GoodsSrv interface {
	List(ctx context.Context, request *gpb.GoodsFilterRequest) (*gpb.GoodsListResponse, error)
}

type goodsService struct {
	data data.DataFactory
}

func (gs *goodsService) List(ctx context.Context, request *gpb.GoodsFilterRequest) (*gpb.GoodsListResponse, error) {
	return gs.data.Goods().GoodsList(ctx, request)
}

func NewGoods(data data.DataFactory) *goodsService {
	return &goodsService{data: data}
}

var _ GoodsSrv = &goodsService{}
