package v1

import (
	"context"
	"strings"

	v1 "goshop/app/goods/srv/internal/data/v1"
	"goshop/app/goods/srv/internal/domain/do"
	"goshop/app/pkg/code"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"
)

type CategorySrv interface {
	ListAll(ctx context.Context, orderBy []string) (*do.CategoryDOList, error)
	Get(ctx context.Context, id uint64) (*do.CategoryDO, error)
	Create(ctx context.Context, category *do.CategoryDO) (*do.CategoryDO, error)
	Update(ctx context.Context, category *do.CategoryDO) error
	Delete(ctx context.Context, id uint64) error
}

type BrandSrv interface {
	List(ctx context.Context, meta metav1.ListMeta, orderBy []string) (*do.BrandsDOList, error)
	Create(ctx context.Context, brand *do.BrandsDO) (*do.BrandsDO, error)
	Update(ctx context.Context, brand *do.BrandsDO) error
	Delete(ctx context.Context, id uint64) error
}

type BannerSrv interface {
	List(ctx context.Context, meta metav1.ListMeta, orderBy []string) (*do.BannerList, error)
	Create(ctx context.Context, banner *do.BannerDO) (*do.BannerDO, error)
	Update(ctx context.Context, banner *do.BannerDO) error
	Delete(ctx context.Context, id uint64) error
}

type CategoryBrandSrv interface {
	List(ctx context.Context, meta metav1.ListMeta, orderBy []string) (*do.GoodsCategoryBrandList, error)
	ListByCategory(ctx context.Context, categoryID uint64, orderBy []string) (*do.GoodsCategoryBrandList, error)
	Create(ctx context.Context, relation *do.GoodsCategoryBrandDO) (*do.GoodsCategoryBrandDO, error)
	Update(ctx context.Context, relation *do.GoodsCategoryBrandDO) error
	Delete(ctx context.Context, id uint64) error
}

type categoryService struct {
	data v1.DataFactory
}

func newCategories(srv *service) *categoryService {
	return &categoryService{data: srv.data}
}

func (s *categoryService) ListAll(ctx context.Context, orderBy []string) (*do.CategoryDOList, error) {
	return s.data.Categories().ListAll(ctx, orderBy)
}

func (s *categoryService) Get(ctx context.Context, id uint64) (*do.CategoryDO, error) {
	if id == 0 {
		return nil, errors.WithCode(code.ErrCategoryNotFound, "category not found")
	}
	return s.data.Categories().Get(ctx, id)
}

func (s *categoryService) Create(ctx context.Context, category *do.CategoryDO) (*do.CategoryDO, error) {
	if err := validateCategory(category, false); err != nil {
		return nil, err
	}
	if category.ParentCategoryID > 0 {
		if _, err := s.data.Categories().Get(ctx, uint64(category.ParentCategoryID)); err != nil {
			return nil, err
		}
	}
	if err := s.data.Categories().Create(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *categoryService) Update(ctx context.Context, category *do.CategoryDO) error {
	if err := validateCategory(category, true); err != nil {
		return err
	}
	if _, err := s.data.Categories().Get(ctx, uint64(category.ID)); err != nil {
		return err
	}
	if category.ParentCategoryID > 0 {
		if _, err := s.data.Categories().Get(ctx, uint64(category.ParentCategoryID)); err != nil {
			return err
		}
	}
	return s.data.Categories().Update(ctx, category)
}

func (s *categoryService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return errors.WithCode(code.ErrCategoryNotFound, "category not found")
	}
	category, err := s.data.Categories().Get(ctx, id)
	if err != nil {
		return err
	}
	if len(category.SubCategory) > 0 {
		return errors.WithCode(code.ErrGoodsInvalid, "category has sub categories")
	}
	return s.data.Categories().Delete(ctx, id)
}

type brandService struct {
	data v1.DataFactory
}

func newBrands(srv *service) *brandService {
	return &brandService{data: srv.data}
}

func (s *brandService) List(ctx context.Context, meta metav1.ListMeta, orderBy []string) (*do.BrandsDOList, error) {
	return s.data.Brands().List(ctx, meta, orderBy)
}

func (s *brandService) Create(ctx context.Context, brand *do.BrandsDO) (*do.BrandsDO, error) {
	if err := validateBrand(brand, false); err != nil {
		return nil, err
	}
	if err := s.data.Brands().Create(ctx, nil, brand); err != nil {
		return nil, err
	}
	return brand, nil
}

func (s *brandService) Update(ctx context.Context, brand *do.BrandsDO) error {
	if err := validateBrand(brand, true); err != nil {
		return err
	}
	if _, err := s.data.Brands().Get(ctx, uint64(brand.ID)); err != nil {
		return err
	}
	return s.data.Brands().Update(ctx, nil, brand)
}

func (s *brandService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return errors.WithCode(code.ErrBrandNotFound, "brand not found")
	}
	if _, err := s.data.Brands().Get(ctx, id); err != nil {
		return err
	}
	return s.data.Brands().Delete(ctx, id)
}

type bannerService struct {
	data v1.DataFactory
}

func newBanners(srv *service) *bannerService {
	return &bannerService{data: srv.data}
}

func (s *bannerService) List(ctx context.Context, meta metav1.ListMeta, orderBy []string) (*do.BannerList, error) {
	return s.data.Banners().List(ctx, meta, orderBy)
}

func (s *bannerService) Create(ctx context.Context, banner *do.BannerDO) (*do.BannerDO, error) {
	if err := validateBanner(banner, false); err != nil {
		return nil, err
	}
	if err := s.data.Banners().Create(ctx, nil, banner); err != nil {
		return nil, err
	}
	return banner, nil
}

func (s *bannerService) Update(ctx context.Context, banner *do.BannerDO) error {
	if err := validateBanner(banner, true); err != nil {
		return err
	}
	return s.data.Banners().Update(ctx, nil, banner)
}

func (s *bannerService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return errors.WithCode(code.ErrBannerNotFound, "banner not found")
	}
	return s.data.Banners().Delete(ctx, id)
}

func validateCategory(category *do.CategoryDO, requireID bool) error {
	if category == nil {
		return errors.WithCode(code.ErrGoodsInvalid, "category is required")
	}
	if requireID && category.ID <= 0 {
		return errors.WithCode(code.ErrCategoryNotFound, "category not found")
	}
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" || category.Level <= 0 {
		return errors.WithCode(code.ErrGoodsInvalid, "category name and level are required")
	}
	if category.ParentCategoryID == category.ID && category.ID > 0 {
		return errors.WithCode(code.ErrGoodsInvalid, "category parent cannot be itself")
	}
	return nil
}

type categoryBrandService struct {
	data v1.DataFactory
}

func newCategoryBrands(srv *service) *categoryBrandService {
	return &categoryBrandService{data: srv.data}
}

func (s *categoryBrandService) List(ctx context.Context, meta metav1.ListMeta, orderBy []string) (*do.GoodsCategoryBrandList, error) {
	return s.data.CategoryBrands().List(ctx, meta, orderBy)
}

func (s *categoryBrandService) ListByCategory(ctx context.Context, categoryID uint64, orderBy []string) (*do.GoodsCategoryBrandList, error) {
	if categoryID == 0 {
		return &do.GoodsCategoryBrandList{}, nil
	}
	if _, err := s.data.Categories().Get(ctx, categoryID); err != nil {
		return nil, err
	}
	return s.data.CategoryBrands().ListByCategory(ctx, categoryID, orderBy)
}

func (s *categoryBrandService) Create(ctx context.Context, relation *do.GoodsCategoryBrandDO) (*do.GoodsCategoryBrandDO, error) {
	if err := validateCategoryBrand(relation, false); err != nil {
		return nil, err
	}
	category, err := s.data.Categories().Get(ctx, uint64(relation.CategoryID))
	if err != nil {
		return nil, err
	}
	brand, err := s.data.Brands().Get(ctx, uint64(relation.BrandsID))
	if err != nil {
		return nil, err
	}
	if err := s.data.CategoryBrands().Create(ctx, nil, relation); err != nil {
		return nil, err
	}
	relation.Category = *category
	relation.Brands = *brand
	return relation, nil
}

func (s *categoryBrandService) Update(ctx context.Context, relation *do.GoodsCategoryBrandDO) error {
	if err := validateCategoryBrand(relation, true); err != nil {
		return err
	}
	if _, err := s.data.Categories().Get(ctx, uint64(relation.CategoryID)); err != nil {
		return err
	}
	if _, err := s.data.Brands().Get(ctx, uint64(relation.BrandsID)); err != nil {
		return err
	}
	return s.data.CategoryBrands().Update(ctx, nil, relation)
}

func (s *categoryBrandService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return errors.WithCode(code.ErrCategoryBrandNotFound, "category brand relation not found")
	}
	return s.data.CategoryBrands().Delete(ctx, id)
}

func validateCategoryBrand(relation *do.GoodsCategoryBrandDO, requireID bool) error {
	if relation == nil {
		return errors.WithCode(code.ErrGoodsInvalid, "category brand relation is required")
	}
	if requireID && relation.ID <= 0 {
		return errors.WithCode(code.ErrCategoryBrandNotFound, "category brand relation not found")
	}
	if relation.CategoryID <= 0 || relation.BrandsID <= 0 {
		return errors.WithCode(code.ErrGoodsInvalid, "category_id and brand_id are required")
	}
	return nil
}

func validateBrand(brand *do.BrandsDO, requireID bool) error {
	if brand == nil {
		return errors.WithCode(code.ErrGoodsInvalid, "brand is required")
	}
	if requireID && brand.ID <= 0 {
		return errors.WithCode(code.ErrBrandNotFound, "brand not found")
	}
	brand.Name = strings.TrimSpace(brand.Name)
	brand.Logo = strings.TrimSpace(brand.Logo)
	if brand.Name == "" {
		return errors.WithCode(code.ErrGoodsInvalid, "brand name is required")
	}
	return nil
}

func validateBanner(banner *do.BannerDO, requireID bool) error {
	if banner == nil {
		return errors.WithCode(code.ErrGoodsInvalid, "banner is required")
	}
	if requireID && banner.ID <= 0 {
		return errors.WithCode(code.ErrBannerNotFound, "banner not found")
	}
	banner.Image = strings.TrimSpace(banner.Image)
	banner.Url = strings.TrimSpace(banner.Url)
	if banner.Image == "" || banner.Url == "" {
		return errors.WithCode(code.ErrGoodsInvalid, "banner image and url are required")
	}
	return nil
}

var _ CategorySrv = &categoryService{}
var _ BrandSrv = &brandService{}
var _ BannerSrv = &bannerService{}
var _ CategoryBrandSrv = &categoryBrandService{}
