package v1

import "gorm.io/gorm"

// 抽象工厂
type DataFactory interface {
	Goods() GoodsStore
	Categories() CategoryStore
	Brands() BrandsStore
	Banners() BannerStore
	CategoryBrands() GoodsCategoryBrandStore

	Begin() *gorm.DB
}
