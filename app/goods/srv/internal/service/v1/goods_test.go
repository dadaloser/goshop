package v1

import (
	"context"
	"testing"

	datav1 "goshop/app/goods/srv/internal/data/v1"
	searchv1 "goshop/app/goods/srv/internal/data_search/v1"
	"goshop/app/goods/srv/internal/domain/do"
	"goshop/app/goods/srv/internal/domain/dto"
	"goshop/app/pkg/code"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"

	"gorm.io/gorm"
)

func TestCreateRejectsInvalidGoods(t *testing.T) {
	tests := []struct {
		name  string
		goods *dto.GoodsDTO
	}{
		{
			name: "nil goods",
		},
		{
			name: "missing category",
			goods: func() *dto.GoodsDTO {
				goods := validGoodsDTO()
				goods.CategoryID = 0
				return goods
			}(),
		},
		{
			name: "missing brand",
			goods: func() *dto.GoodsDTO {
				goods := validGoodsDTO()
				goods.CategoryID = 1
				goods.BrandsID = 0
				return goods
			}(),
		},
		{
			name: "missing name",
			goods: func() *dto.GoodsDTO {
				goods := validGoodsDTO()
				goods.CategoryID = 1
				goods.Name = " "
				return goods
			}(),
		},
		{
			name: "negative price",
			goods: func() *dto.GoodsDTO {
				goods := validGoodsDTO()
				goods.CategoryID = 1
				goods.ShopPrice = -1
				return goods
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &goodsService{}
			err := svc.Create(context.Background(), tt.goods)
			if !errors.IsCode(err, code.ErrGoodsInvalid) {
				t.Fatalf("Create() error = %v, want code %d", err, code.ErrGoodsInvalid)
			}
		})
	}
}

func TestUpdateRejectsInvalidGoods(t *testing.T) {
	tests := []struct {
		name  string
		goods *dto.GoodsDTO
	}{
		{
			name: "nil goods",
		},
		{
			name:  "missing id",
			goods: validGoodsDTO(),
		},
		{
			name: "missing goods sn",
			goods: func() *dto.GoodsDTO {
				goods := validGoodsDTO()
				goods.ID = 1
				goods.GoodsSn = " "
				return goods
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &goodsService{}
			err := svc.Update(context.Background(), tt.goods)
			if !errors.IsCode(err, code.ErrGoodsInvalid) {
				t.Fatalf("Update() error = %v, want code %d", err, code.ErrGoodsInvalid)
			}
		})
	}
}

func TestDeleteRejectsMissingGoodsID(t *testing.T) {
	svc := &goodsService{}

	err := svc.Delete(context.Background(), 0)
	if !errors.IsCode(err, code.ErrGoodsInvalid) {
		t.Fatalf("Delete() error = %v, want code %d", err, code.ErrGoodsInvalid)
	}
}

func TestUpdateChecksGoodsExistsBeforeReferences(t *testing.T) {
	brandLookups := 0
	svc := &goodsService{
		data: fakeGoodsDataFactory{
			goods: fakeGoodsStore{
				get: func(context.Context, uint64) (*do.GoodsDO, error) {
					return nil, errors.WithCode(code.ErrGoodsNotFound, "goods not found")
				},
			},
			brands: fakeBrandStore{
				get: func(context.Context, uint64) (*do.BrandsDO, error) {
					brandLookups++
					return &do.BrandsDO{}, nil
				},
			},
		},
	}

	goods := validGoodsDTO()
	goods.ID = 1
	err := svc.Update(context.Background(), goods)
	if !errors.IsCode(err, code.ErrGoodsNotFound) {
		t.Fatalf("Update() error = %v, want code %d", err, code.ErrGoodsNotFound)
	}
	if brandLookups != 0 {
		t.Fatalf("Update() brand lookups = %d, want 0", brandLookups)
	}
}

func validGoodsDTO() *dto.GoodsDTO {
	return &dto.GoodsDTO{
		GoodsDO: do.GoodsDO{
			CategoryID:      1,
			BrandsID:        1,
			Name:            "goods",
			GoodsSn:         "goods-sn",
			MarketPrice:     10,
			ShopPrice:       8,
			GoodsBrief:      "brief",
			GoodsFrontImage: "front.jpg",
		},
	}
}

type fakeGoodsDataFactory struct {
	goods          datav1.GoodsStore
	categories     datav1.CategoryStore
	brands         datav1.BrandsStore
	banners        datav1.BannerStore
	categoryBrands datav1.GoodsCategoryBrandStore
}

func (f fakeGoodsDataFactory) Goods() datav1.GoodsStore {
	return f.goods
}

func (f fakeGoodsDataFactory) Categories() datav1.CategoryStore {
	return f.categories
}

func (f fakeGoodsDataFactory) Brands() datav1.BrandsStore {
	return f.brands
}

func (f fakeGoodsDataFactory) Banners() datav1.BannerStore {
	return f.banners
}

func (f fakeGoodsDataFactory) CategoryBrands() datav1.GoodsCategoryBrandStore {
	return f.categoryBrands
}

func (fakeGoodsDataFactory) Begin() *gorm.DB {
	return &gorm.DB{}
}

type fakeGoodsStore struct {
	get func(context.Context, uint64) (*do.GoodsDO, error)
}

func (f fakeGoodsStore) Get(ctx context.Context, id uint64) (*do.GoodsDO, error) {
	if f.get != nil {
		return f.get(ctx, id)
	}
	return &do.GoodsDO{}, nil
}

func (fakeGoodsStore) ListByIDs(context.Context, []uint64, []string) (*do.GoodsDOList, error) {
	return nil, nil
}

func (fakeGoodsStore) List(context.Context, []string, metav1.ListMeta) (*do.GoodsDOList, error) {
	return nil, nil
}

func (fakeGoodsStore) Create(context.Context, *do.GoodsDO) error {
	return nil
}

func (fakeGoodsStore) CreateInTxn(context.Context, *gorm.DB, *do.GoodsDO) error {
	return nil
}

func (fakeGoodsStore) Update(context.Context, *do.GoodsDO) error {
	return nil
}

func (fakeGoodsStore) UpdateInTxn(context.Context, *gorm.DB, *do.GoodsDO) error {
	return nil
}

func (fakeGoodsStore) Delete(context.Context, uint64) error {
	return nil
}

func (fakeGoodsStore) DeleteInTxn(context.Context, *gorm.DB, uint64) error {
	return nil
}

func (fakeGoodsStore) Begin() *gorm.DB {
	return &gorm.DB{}
}

type fakeBrandStore struct {
	get func(context.Context, uint64) (*do.BrandsDO, error)
}

func (fakeBrandStore) List(context.Context, metav1.ListMeta, []string) (*do.BrandsDOList, error) {
	return nil, nil
}

func (fakeBrandStore) Create(context.Context, *gorm.DB, *do.BrandsDO) error {
	return nil
}

func (fakeBrandStore) Update(context.Context, *gorm.DB, *do.BrandsDO) error {
	return nil
}

func (fakeBrandStore) Delete(context.Context, uint64) error {
	return nil
}

func (f fakeBrandStore) Get(ctx context.Context, id uint64) (*do.BrandsDO, error) {
	if f.get != nil {
		return f.get(ctx, id)
	}
	return &do.BrandsDO{}, nil
}

type fakeSearchFactory struct {
	goods searchv1.GoodsStore
}

func (f fakeSearchFactory) Goods() searchv1.GoodsStore {
	return f.goods
}

type fakeSearchGoodsStore struct{}

func (fakeSearchGoodsStore) Create(context.Context, *do.GoodsSearchDO) error {
	return nil
}

func (fakeSearchGoodsStore) Delete(context.Context, uint64) error {
	return nil
}

func (fakeSearchGoodsStore) Update(context.Context, *do.GoodsSearchDO) error {
	return nil
}

func (fakeSearchGoodsStore) Search(context.Context, *searchv1.GoodsFilterRequest) (*do.GoodsSearchDOList, error) {
	return nil, nil
}

var _ datav1.DataFactory = fakeGoodsDataFactory{}
var _ datav1.GoodsStore = fakeGoodsStore{}
var _ datav1.BrandsStore = fakeBrandStore{}
var _ searchv1.SearchFactory = fakeSearchFactory{}
var _ searchv1.GoodsStore = fakeSearchGoodsStore{}
