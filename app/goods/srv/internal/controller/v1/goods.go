package v1

import (
	"context"
	"goshop/app/goods/srv/internal/domain/do"
	"goshop/app/goods/srv/internal/domain/dto"
	bgorm "goshop/app/pkg/gorm"
	v12 "goshop/pkg/common/meta/v1"

	proto "goshop/api/goods/v1"
	v1 "goshop/app/goods/srv/internal/service/v1"
	"goshop/pkg/log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type goodsServer struct {
	proto.UnimplementedGoodsServer
	srv v1.ServiceFactory
}

func ModelToResponse(goods *dto.GoodsDTO) *proto.GoodsInfoResponse {
	return &proto.GoodsInfoResponse{
		Id:              goods.ID,
		CategoryId:      goods.CategoryID,
		Name:            goods.Name,
		GoodsSn:         goods.GoodsSn,
		ClickNum:        goods.ClickNum,
		SoldNum:         goods.SoldNum,
		FavNum:          goods.FavNum,
		MarketPrice:     goods.MarketPrice,
		ShopPrice:       goods.ShopPrice,
		GoodsBrief:      goods.GoodsBrief,
		ShipFree:        goods.ShipFree,
		GoodsFrontImage: goods.GoodsFrontImage,
		IsNew:           goods.IsNew,
		IsHot:           goods.IsHot,
		OnSale:          goods.OnSale,
		DescImages:      goods.DescImages,
		Images:          goods.Images,
		Category: &proto.CategoryBriefInfoResponse{
			Id:   goods.Category.ID,
			Name: goods.Category.Name,
		},
		Brand: &proto.BrandInfoResponse{
			Id:   goods.Brands.ID,
			Name: goods.Brands.Name,
			Logo: goods.Brands.Logo,
		},
	}
}

func (gs *goodsServer) GoodsList(ctx context.Context, request *proto.GoodsFilterRequest) (*proto.GoodsListResponse, error) {
	list, err := gs.srv.Goods().List(ctx, v12.ListMeta{Page: int(request.Pages), PageSize: int(request.PagePerNums)}, request, []string{})
	if err != nil {
		log.Errorf("get goods list error: %v", err.Error())
		return nil, err
	}
	var ret proto.GoodsListResponse
	ret.Total = int32(list.TotalCount)
	for _, item := range list.Items {
		ret.Data = append(ret.Data, ModelToResponse(item))
	}
	return &ret, nil
}

func (gs *goodsServer) BatchGetGoods(ctx context.Context, info *proto.BatchGoodsIdInfo) (*proto.GoodsListResponse, error) {
	var ids []uint64
	for _, id := range info.Id {
		ids = append(ids, uint64(id))
	}
	get, err := gs.srv.Goods().BatchGet(ctx, ids)
	if err != nil {
		return nil, err
	}
	var ret proto.GoodsListResponse
	for _, item := range get {
		ret.Data = append(ret.Data, ModelToResponse(item))
	}
	return &ret, nil
}

func (gs *goodsServer) CreateGoods(ctx context.Context, info *proto.CreateGoodsInfo) (*proto.GoodsInfoResponse, error) {
	goods := createGoodsInfoToDTO(info)
	if err := gs.srv.Goods().Create(ctx, goods); err != nil {
		return nil, err
	}
	created, err := gs.srv.Goods().Get(ctx, uint64(goods.ID))
	if err != nil {
		return nil, err
	}
	return ModelToResponse(created), nil
}

func (gs *goodsServer) DeleteGoods(ctx context.Context, info *proto.DeleteGoodsInfo) (*emptypb.Empty, error) {
	if err := gs.srv.Goods().Delete(ctx, uint64(info.Id)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) UpdateGoods(ctx context.Context, info *proto.CreateGoodsInfo) (*emptypb.Empty, error) {
	if err := gs.srv.Goods().Update(ctx, createGoodsInfoToDTO(info)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) GetGoodsDetail(ctx context.Context, request *proto.GoodInfoRequest) (*proto.GoodsInfoResponse, error) {
	goods, err := gs.srv.Goods().Get(ctx, uint64(request.Id))
	if err != nil {
		return nil, err
	}
	return ModelToResponse(goods), nil
}

func (gs *goodsServer) GetAllCategorysList(ctx context.Context, empty *emptypb.Empty) (*proto.CategoryListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "category list is not implemented yet")
}

func (gs *goodsServer) GetSubCategory(ctx context.Context, request *proto.CategoryListRequest) (*proto.SubCategoryListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "sub category list is not implemented yet")
}

func (gs *goodsServer) CreateCategory(ctx context.Context, request *proto.CategoryInfoRequest) (*proto.CategoryInfoResponse, error) {
	return nil, status.Error(codes.Unimplemented, "create category is not implemented yet")
}

func (gs *goodsServer) DeleteCategory(ctx context.Context, request *proto.DeleteCategoryRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "delete category is not implemented yet")
}

func (gs *goodsServer) UpdateCategory(ctx context.Context, request *proto.CategoryInfoRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "update category is not implemented yet")
}

func (gs *goodsServer) BrandList(ctx context.Context, request *proto.BrandFilterRequest) (*proto.BrandListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "brand list is not implemented yet")
}

func (gs *goodsServer) CreateBrand(ctx context.Context, request *proto.BrandRequest) (*proto.BrandInfoResponse, error) {
	return nil, status.Error(codes.Unimplemented, "create brand is not implemented yet")
}

func (gs *goodsServer) DeleteBrand(ctx context.Context, request *proto.BrandRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "delete brand is not implemented yet")
}

func (gs *goodsServer) UpdateBrand(ctx context.Context, request *proto.BrandRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "update brand is not implemented yet")
}

func (gs *goodsServer) BannerList(ctx context.Context, empty *emptypb.Empty) (*proto.BannerListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "banner list is not implemented yet")
}

func (gs *goodsServer) CreateBanner(ctx context.Context, request *proto.BannerRequest) (*proto.BannerResponse, error) {
	return nil, status.Error(codes.Unimplemented, "create banner is not implemented yet")
}

func (gs *goodsServer) DeleteBanner(ctx context.Context, request *proto.BannerRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "delete banner is not implemented yet")
}

func (gs *goodsServer) UpdateBanner(ctx context.Context, request *proto.BannerRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "update banner is not implemented yet")
}

func (gs *goodsServer) CategoryBrandList(ctx context.Context, request *proto.CategoryBrandFilterRequest) (*proto.CategoryBrandListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "category brand list is not implemented yet")
}

func (gs *goodsServer) GetCategoryBrandList(ctx context.Context, request *proto.CategoryInfoRequest) (*proto.BrandListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "get category brand list is not implemented yet")
}

func (gs *goodsServer) CreateCategoryBrand(ctx context.Context, request *proto.CategoryBrandRequest) (*proto.CategoryBrandResponse, error) {
	return nil, status.Error(codes.Unimplemented, "create category brand is not implemented yet")
}

func (gs *goodsServer) DeleteCategoryBrand(ctx context.Context, request *proto.CategoryBrandRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "delete category brand is not implemented yet")
}

func (gs *goodsServer) UpdateCategoryBrand(ctx context.Context, request *proto.CategoryBrandRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "update category brand is not implemented yet")
}

func NewGoodsServer(srv v1.ServiceFactory) *goodsServer {
	return &goodsServer{srv: srv}
}

func createGoodsInfoToDTO(info *proto.CreateGoodsInfo) *dto.GoodsDTO {
	return &dto.GoodsDTO{
		GoodsDO: do.GoodsDO{
			BaseModel:       bgorm.BaseModel{ID: info.Id},
			CategoryID:      info.CategoryId,
			BrandsID:        info.BrandId,
			OnSale:          info.OnSale,
			ShipFree:        info.ShipFree,
			IsNew:           info.IsNew,
			IsHot:           info.IsHot,
			Name:            info.Name,
			GoodsSn:         info.GoodsSn,
			MarketPrice:     info.MarketPrice,
			ShopPrice:       info.ShopPrice,
			GoodsBrief:      info.GoodsBrief,
			Images:          do.GormList(info.Images),
			DescImages:      do.GormList(info.DescImages),
			GoodsFrontImage: info.GoodsFrontImage,
		},
	}
}
