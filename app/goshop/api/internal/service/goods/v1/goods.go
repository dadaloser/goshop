package v1

import (
	"context"
	gpb "goshop/api/goods/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"
)

//服务接口要与app/goods/srv/internal/service/v1/goods.go的GoodsSrv接口保持一致

type GoodsSrv interface {
	List(ctx context.Context, request *gpb.GoodsFilterRequest) (*gpb.GoodsListResponse, error)
}

type goodsService struct {
	data data.DataFactory
}

func (gs *goodsService) List(ctx context.Context, request *gpb.GoodsFilterRequest) (*gpb.GoodsListResponse, error) {
	if gs == nil || gs.data == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "goods data client is not initialized")
	}
	if request == nil {
		return nil, errors.WithCode(code.ErrGoodsInvalid, "goods filter request is required")
	}

	client := gs.data.Goods()
	if client == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "goods grpc client is not initialized")
	}
	resp, err := client.GoodsList(ctx, request)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.WithCode(code.ErrConnectGRPC, "goods grpc response is empty")
	}
	return resp, nil
}

func NewGoods(data data.DataFactory) *goodsService {
	return &goodsService{data: data}
}

var _ GoodsSrv = &goodsService{}
