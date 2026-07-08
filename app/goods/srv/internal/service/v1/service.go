package v1

import (
	v1 "goshop/app/goods/srv/internal/data/v1"
	v12 "goshop/app/goods/srv/internal/data_search/v1"
)

type ServiceFactory interface {
	Goods() GoodsSrv
	Categories() CategorySrv
	Brands() BrandSrv
	Banners() BannerSrv
	CategoryBrands() CategoryBrandSrv
}

type service struct {
	data       v1.DataFactory
	dataSearch v12.SearchFactory
}

func NewService(store v1.DataFactory, dataSearch v12.SearchFactory) *service {
	return &service{data: store, dataSearch: dataSearch}
}

var _ ServiceFactory = &service{}

func (s *service) Goods() GoodsSrv {
	return newGoods(s)
}

func (s *service) Categories() CategorySrv {
	return newCategories(s)
}

func (s *service) Brands() BrandSrv {
	return newBrands(s)
}

func (s *service) Banners() BannerSrv {
	return newBanners(s)
}

func (s *service) CategoryBrands() CategoryBrandSrv {
	return newCategoryBrands(s)
}
