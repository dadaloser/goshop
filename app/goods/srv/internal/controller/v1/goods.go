package v1

import (
	"context"
	"goshop/app/goods/srv/internal/domain/do"
	"goshop/app/goods/srv/internal/domain/dto"
	bgorm "goshop/app/pkg/gorm"
	v12 "goshop/pkg/common/meta/v1"
	"goshop/pkg/money"

	proto "goshop/api/goods/v1"
	v1 "goshop/app/goods/srv/internal/service/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
	"goshop/pkg/log"

	"google.golang.org/protobuf/types/known/emptypb"
)

type goodsServer struct {
	proto.UnimplementedGoodsServer
	srv v1.ServiceFactory
}

func ModelToResponse(goods *dto.GoodsDTO) *proto.GoodsInfoResponse {
	goods.SyncLegacyMoneyFields()
	return &proto.GoodsInfoResponse{
		Id:              goods.ID,
		CategoryId:      goods.CategoryID,
		Name:            goods.Name,
		GoodsSn:         goods.GoodsSn,
		ClickNum:        goods.ClickNum,
		SoldNum:         goods.SoldNum,
		FavNum:          goods.FavNum,
		MarketPrice:     goods.MarketPrice,
		MarketPriceFen:  goods.MarketPriceFen,
		ShopPrice:       goods.ShopPrice,
		ShopPriceFen:    goods.ShopPriceFen,
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
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "goods filter request is required")
	}

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
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "batch goods request is required")
	}

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
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "goods request is required")
	}

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
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "delete goods request is required")
	}

	if err := gs.srv.Goods().Delete(ctx, uint64(info.Id)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) UpdateGoods(ctx context.Context, info *proto.CreateGoodsInfo) (*emptypb.Empty, error) {
	if info == nil {
		return nil, errors.WithCode(code2.ErrValidation, "goods request is required")
	}

	if err := gs.srv.Goods().Update(ctx, createGoodsInfoToDTO(info)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) GetGoodsDetail(ctx context.Context, request *proto.GoodInfoRequest) (*proto.GoodsInfoResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "goods detail request is required")
	}

	goods, err := gs.srv.Goods().Get(ctx, uint64(request.Id))
	if err != nil {
		return nil, err
	}
	return ModelToResponse(goods), nil
}

func (gs *goodsServer) GetAllCategorysList(ctx context.Context, empty *emptypb.Empty) (*proto.CategoryListResponse, error) {
	list, err := gs.srv.Categories().ListAll(ctx, []string{})
	if err != nil {
		return nil, err
	}

	ret := &proto.CategoryListResponse{Total: int32(list.TotalCount)}
	for _, item := range list.Items {
		ret.Data = append(ret.Data, categoryToResponse(item))
	}
	return ret, nil
}

func (gs *goodsServer) GetSubCategory(ctx context.Context, request *proto.CategoryListRequest) (*proto.SubCategoryListResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "category request is required")
	}

	category, err := gs.srv.Categories().Get(ctx, uint64(request.GetId()))
	if err != nil {
		return nil, err
	}

	ret := &proto.SubCategoryListResponse{
		Info:  categoryToResponse(category),
		Total: int32(len(category.SubCategory)),
	}
	for _, item := range category.SubCategory {
		ret.SubCategorys = append(ret.SubCategorys, categoryToResponse(item))
	}
	return ret, nil
}

func (gs *goodsServer) CreateCategory(ctx context.Context, request *proto.CategoryInfoRequest) (*proto.CategoryInfoResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "category request is required")
	}

	category, err := gs.srv.Categories().Create(ctx, categoryRequestToDO(request))
	if err != nil {
		return nil, err
	}
	return categoryToResponse(category), nil
}

func (gs *goodsServer) DeleteCategory(ctx context.Context, request *proto.DeleteCategoryRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "delete category request is required")
	}

	if err := gs.srv.Categories().Delete(ctx, uint64(request.GetId())); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) UpdateCategory(ctx context.Context, request *proto.CategoryInfoRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "category request is required")
	}

	if err := gs.srv.Categories().Update(ctx, categoryRequestToDO(request)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) BrandList(ctx context.Context, request *proto.BrandFilterRequest) (*proto.BrandListResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "brand filter request is required")
	}

	list, err := gs.srv.Brands().List(ctx, v12.ListMeta{Page: int(request.GetPages()), PageSize: int(request.GetPagePerNums())}, []string{})
	if err != nil {
		return nil, err
	}

	ret := &proto.BrandListResponse{Total: int32(list.TotalCount)}
	for _, item := range list.Items {
		ret.Data = append(ret.Data, brandToResponse(item))
	}
	return ret, nil
}

func (gs *goodsServer) CreateBrand(ctx context.Context, request *proto.BrandRequest) (*proto.BrandInfoResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "brand request is required")
	}

	brand, err := gs.srv.Brands().Create(ctx, brandRequestToDO(request))
	if err != nil {
		return nil, err
	}
	return brandToResponse(brand), nil
}

func (gs *goodsServer) DeleteBrand(ctx context.Context, request *proto.BrandRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "brand request is required")
	}

	if err := gs.srv.Brands().Delete(ctx, uint64(request.GetId())); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) UpdateBrand(ctx context.Context, request *proto.BrandRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "brand request is required")
	}

	if err := gs.srv.Brands().Update(ctx, brandRequestToDO(request)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) BannerList(ctx context.Context, empty *emptypb.Empty) (*proto.BannerListResponse, error) {
	list, err := gs.srv.Banners().List(ctx, v12.ListMeta{PageSize: 100}, []string{"`index` asc", "id asc"})
	if err != nil {
		return nil, err
	}

	ret := &proto.BannerListResponse{Total: int32(list.TotalCount)}
	for _, item := range list.Items {
		ret.Data = append(ret.Data, bannerToResponse(item))
	}
	return ret, nil
}

func (gs *goodsServer) CreateBanner(ctx context.Context, request *proto.BannerRequest) (*proto.BannerResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "banner request is required")
	}

	banner, err := gs.srv.Banners().Create(ctx, bannerRequestToDO(request))
	if err != nil {
		return nil, err
	}
	return bannerToResponse(banner), nil
}

func (gs *goodsServer) DeleteBanner(ctx context.Context, request *proto.BannerRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "banner request is required")
	}

	if err := gs.srv.Banners().Delete(ctx, uint64(request.GetId())); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) UpdateBanner(ctx context.Context, request *proto.BannerRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "banner request is required")
	}

	if err := gs.srv.Banners().Update(ctx, bannerRequestToDO(request)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) CategoryBrandList(ctx context.Context, request *proto.CategoryBrandFilterRequest) (*proto.CategoryBrandListResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "category brand filter request is required")
	}

	list, err := gs.srv.CategoryBrands().List(ctx, v12.ListMeta{Page: int(request.GetPages()), PageSize: int(request.GetPagePerNums())}, []string{})
	if err != nil {
		return nil, err
	}

	ret := &proto.CategoryBrandListResponse{Total: int32(list.TotalCount)}
	for _, item := range list.Items {
		ret.Data = append(ret.Data, categoryBrandToResponse(item))
	}
	return ret, nil
}

func (gs *goodsServer) GetCategoryBrandList(ctx context.Context, request *proto.CategoryInfoRequest) (*proto.BrandListResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "category request is required")
	}

	list, err := gs.srv.CategoryBrands().ListByCategory(ctx, uint64(request.GetId()), []string{})
	if err != nil {
		return nil, err
	}

	ret := &proto.BrandListResponse{Total: int32(list.TotalCount)}
	for _, item := range list.Items {
		ret.Data = append(ret.Data, brandToResponse(&item.Brands))
	}
	return ret, nil
}

func (gs *goodsServer) CreateCategoryBrand(ctx context.Context, request *proto.CategoryBrandRequest) (*proto.CategoryBrandResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "category brand request is required")
	}

	relation, err := gs.srv.CategoryBrands().Create(ctx, categoryBrandRequestToDO(request))
	if err != nil {
		return nil, err
	}
	return categoryBrandToResponse(relation), nil
}

func (gs *goodsServer) DeleteCategoryBrand(ctx context.Context, request *proto.CategoryBrandRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "category brand request is required")
	}

	if err := gs.srv.CategoryBrands().Delete(ctx, uint64(request.GetId())); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (gs *goodsServer) UpdateCategoryBrand(ctx context.Context, request *proto.CategoryBrandRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "category brand request is required")
	}

	if err := gs.srv.CategoryBrands().Update(ctx, categoryBrandRequestToDO(request)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func NewGoodsServer(srv v1.ServiceFactory) *goodsServer {
	return &goodsServer{srv: srv}
}

func createGoodsInfoToDTO(info *proto.CreateGoodsInfo) *dto.GoodsDTO {
	marketPriceFen := info.GetMarketPriceFen()
	if marketPriceFen == 0 && info.GetMarketPrice() != 0 {
		marketPriceFen = money.FromLegacyFloat32Yuan(info.GetMarketPrice()).Int64()
	}
	shopPriceFen := info.GetShopPriceFen()
	if shopPriceFen == 0 && info.GetShopPrice() != 0 {
		shopPriceFen = money.FromLegacyFloat32Yuan(info.GetShopPrice()).Int64()
	}
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
			MarketPrice:     money.NewFen(marketPriceFen).Float32Yuan(),
			MarketPriceFen:  marketPriceFen,
			ShopPrice:       money.NewFen(shopPriceFen).Float32Yuan(),
			ShopPriceFen:    shopPriceFen,
			GoodsBrief:      info.GoodsBrief,
			Images:          do.GormList(info.Images),
			DescImages:      do.GormList(info.DescImages),
			GoodsFrontImage: info.GoodsFrontImage,
		},
	}
}

func categoryToResponse(category *do.CategoryDO) *proto.CategoryInfoResponse {
	if category == nil {
		return nil
	}
	return &proto.CategoryInfoResponse{
		Id:             category.ID,
		Name:           category.Name,
		ParentCategory: category.ParentCategoryID,
		Level:          category.Level,
		IsTab:          category.IsTab,
	}
}

func categoryRequestToDO(request *proto.CategoryInfoRequest) *do.CategoryDO {
	return &do.CategoryDO{
		BaseModel:        bgorm.BaseModel{ID: request.GetId()},
		Name:             request.GetName(),
		ParentCategoryID: request.GetParentCategory(),
		Level:            request.GetLevel(),
		IsTab:            request.GetIsTab(),
	}
}

func brandToResponse(brand *do.BrandsDO) *proto.BrandInfoResponse {
	if brand == nil {
		return nil
	}
	return &proto.BrandInfoResponse{
		Id:   brand.ID,
		Name: brand.Name,
		Logo: brand.Logo,
	}
}

func brandRequestToDO(request *proto.BrandRequest) *do.BrandsDO {
	return &do.BrandsDO{
		BaseModel: bgorm.BaseModel{ID: request.GetId()},
		Name:      request.GetName(),
		Logo:      request.GetLogo(),
	}
}

func bannerToResponse(banner *do.BannerDO) *proto.BannerResponse {
	if banner == nil {
		return nil
	}
	return &proto.BannerResponse{
		Id:    banner.ID,
		Index: banner.Index,
		Image: banner.Image,
		Url:   banner.Url,
	}
}

func bannerRequestToDO(request *proto.BannerRequest) *do.BannerDO {
	return &do.BannerDO{
		BaseModel: bgorm.BaseModel{ID: request.GetId()},
		Index:     request.GetIndex(),
		Image:     request.GetImage(),
		Url:       request.GetUrl(),
	}
}

func categoryBrandToResponse(relation *do.GoodsCategoryBrandDO) *proto.CategoryBrandResponse {
	if relation == nil {
		return nil
	}
	return &proto.CategoryBrandResponse{
		Id:       relation.ID,
		Brand:    brandToResponse(&relation.Brands),
		Category: categoryToResponse(&relation.Category),
	}
}

func categoryBrandRequestToDO(request *proto.CategoryBrandRequest) *do.GoodsCategoryBrandDO {
	return &do.GoodsCategoryBrandDO{
		BaseModel:  bgorm.BaseModel{ID: request.GetId()},
		CategoryID: request.GetCategoryId(),
		BrandsID:   request.GetBrandId(),
	}
}
