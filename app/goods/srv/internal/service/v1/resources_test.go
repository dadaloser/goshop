package v1

import (
	"context"
	"testing"

	"goshop/app/goods/srv/internal/domain/do"
	"goshop/app/pkg/code"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"

	"gorm.io/gorm"
)

func TestCategoryDeleteRejectsCategoryWithChildren(t *testing.T) {
	deleteCalls := 0
	svc := &categoryService{
		data: fakeGoodsDataFactory{
			goods:          fakeGoodsStore{},
			categoryBrands: fakeCategoryBrandStore{},
			categories: fakeCategoryStore{
				get: func(context.Context, uint64) (*do.CategoryDO, error) {
					return &do.CategoryDO{
						SubCategory: []*do.CategoryDO{
							{Name: "child"},
						},
					}, nil
				},
				delete: func(context.Context, uint64) error {
					deleteCalls++
					return nil
				},
			},
		},
	}

	err := svc.Delete(context.Background(), 10)
	if !errors.IsCode(err, code.ErrGoodsInvalid) {
		t.Fatalf("Delete() error = %v, want code %d", err, code.ErrGoodsInvalid)
	}
	if deleteCalls != 0 {
		t.Fatalf("Delete() data delete calls = %d, want 0", deleteCalls)
	}
}

func TestCategoryDeleteRejectsCategoryWithGoods(t *testing.T) {
	deleteCalls := 0
	svc := &categoryService{
		data: fakeGoodsDataFactory{
			categories: fakeCategoryStore{
				get: func(context.Context, uint64) (*do.CategoryDO, error) {
					return &do.CategoryDO{}, nil
				},
				delete: func(context.Context, uint64) error {
					deleteCalls++
					return nil
				},
			},
			goods: fakeGoodsStore{
				countByCategory: func(context.Context, uint64) (int64, error) {
					return 2, nil
				},
			},
			categoryBrands: fakeCategoryBrandStore{},
		},
	}

	err := svc.Delete(context.Background(), 10)
	if !errors.IsCode(err, code.ErrGoodsInvalid) {
		t.Fatalf("Delete() error = %v, want code %d", err, code.ErrGoodsInvalid)
	}
	if deleteCalls != 0 {
		t.Fatalf("Delete() data delete calls = %d, want 0", deleteCalls)
	}
}

func TestCategoryDeleteRejectsCategoryBrandRelations(t *testing.T) {
	deleteCalls := 0
	svc := &categoryService{
		data: fakeGoodsDataFactory{
			categories: fakeCategoryStore{
				get: func(context.Context, uint64) (*do.CategoryDO, error) {
					return &do.CategoryDO{}, nil
				},
				delete: func(context.Context, uint64) error {
					deleteCalls++
					return nil
				},
			},
			goods: fakeGoodsStore{},
			categoryBrands: fakeCategoryBrandStore{
				listByCategory: func(context.Context, uint64) (*do.GoodsCategoryBrandList, error) {
					return &do.GoodsCategoryBrandList{TotalCount: 1}, nil
				},
			},
		},
	}

	err := svc.Delete(context.Background(), 10)
	if !errors.IsCode(err, code.ErrGoodsInvalid) {
		t.Fatalf("Delete() error = %v, want code %d", err, code.ErrGoodsInvalid)
	}
	if deleteCalls != 0 {
		t.Fatalf("Delete() data delete calls = %d, want 0", deleteCalls)
	}
}

func TestBrandDeleteRejectsBrandWithGoods(t *testing.T) {
	deleteCalls := 0
	svc := &brandService{
		data: fakeGoodsDataFactory{
			brands: fakeBrandStore{
				get: func(context.Context, uint64) (*do.BrandsDO, error) {
					return &do.BrandsDO{}, nil
				},
				delete: func(context.Context, uint64) error {
					deleteCalls++
					return nil
				},
			},
			goods: fakeGoodsStore{
				countByBrand: func(context.Context, uint64) (int64, error) {
					return 1, nil
				},
			},
			categoryBrands: fakeCategoryBrandStore{},
		},
	}

	err := svc.Delete(context.Background(), 20)
	if !errors.IsCode(err, code.ErrGoodsInvalid) {
		t.Fatalf("Delete() error = %v, want code %d", err, code.ErrGoodsInvalid)
	}
	if deleteCalls != 0 {
		t.Fatalf("Delete() data delete calls = %d, want 0", deleteCalls)
	}
}

func TestBrandDeleteRejectsCategoryBrandRelations(t *testing.T) {
	deleteCalls := 0
	svc := &brandService{
		data: fakeGoodsDataFactory{
			brands: fakeBrandStore{
				get: func(context.Context, uint64) (*do.BrandsDO, error) {
					return &do.BrandsDO{}, nil
				},
				delete: func(context.Context, uint64) error {
					deleteCalls++
					return nil
				},
			},
			goods: fakeGoodsStore{},
			categoryBrands: fakeCategoryBrandStore{
				countByBrand: func(context.Context, uint64) (int64, error) {
					return 1, nil
				},
			},
		},
	}

	err := svc.Delete(context.Background(), 20)
	if !errors.IsCode(err, code.ErrGoodsInvalid) {
		t.Fatalf("Delete() error = %v, want code %d", err, code.ErrGoodsInvalid)
	}
	if deleteCalls != 0 {
		t.Fatalf("Delete() data delete calls = %d, want 0", deleteCalls)
	}
}

func TestCategoryBrandCreateValidatesReferences(t *testing.T) {
	created := false
	svc := &categoryBrandService{
		data: fakeGoodsDataFactory{
			categories: fakeCategoryStore{
				get: func(context.Context, uint64) (*do.CategoryDO, error) {
					return &do.CategoryDO{Name: "category"}, nil
				},
			},
			brands: fakeBrandStore{
				get: func(context.Context, uint64) (*do.BrandsDO, error) {
					return nil, errors.WithCode(code.ErrBrandNotFound, "brand not found")
				},
			},
			categoryBrands: fakeCategoryBrandStore{
				create: func(context.Context, *do.GoodsCategoryBrandDO) error {
					created = true
					return nil
				},
			},
		},
	}

	_, err := svc.Create(context.Background(), &do.GoodsCategoryBrandDO{CategoryID: 1, BrandsID: 2})
	if !errors.IsCode(err, code.ErrBrandNotFound) {
		t.Fatalf("Create() error = %v, want code %d", err, code.ErrBrandNotFound)
	}
	if created {
		t.Fatal("Create() wrote category brand relation before validating brand")
	}
}

type fakeCategoryStore struct {
	get    func(context.Context, uint64) (*do.CategoryDO, error)
	delete func(context.Context, uint64) error
}

func (f fakeCategoryStore) Get(ctx context.Context, id uint64) (*do.CategoryDO, error) {
	if f.get != nil {
		return f.get(ctx, id)
	}
	return &do.CategoryDO{}, nil
}

func (fakeCategoryStore) ListAll(context.Context, []string) (*do.CategoryDOList, error) {
	return nil, nil
}

func (fakeCategoryStore) Create(context.Context, *do.CategoryDO) error {
	return nil
}

func (fakeCategoryStore) Update(context.Context, *do.CategoryDO) error {
	return nil
}

func (f fakeCategoryStore) Delete(ctx context.Context, id uint64) error {
	if f.delete != nil {
		return f.delete(ctx, id)
	}
	return nil
}

type fakeCategoryBrandStore struct {
	listByCategory func(context.Context, uint64) (*do.GoodsCategoryBrandList, error)
	countByBrand   func(context.Context, uint64) (int64, error)
	create         func(context.Context, *do.GoodsCategoryBrandDO) error
}

func (fakeCategoryBrandStore) List(context.Context, metav1.ListMeta, []string) (*do.GoodsCategoryBrandList, error) {
	return nil, nil
}

func (f fakeCategoryBrandStore) ListByCategory(ctx context.Context, categoryID uint64, _ []string) (*do.GoodsCategoryBrandList, error) {
	if f.listByCategory != nil {
		return f.listByCategory(ctx, categoryID)
	}
	return &do.GoodsCategoryBrandList{}, nil
}

func (f fakeCategoryBrandStore) CountByBrand(ctx context.Context, brandID uint64) (int64, error) {
	if f.countByBrand != nil {
		return f.countByBrand(ctx, brandID)
	}
	return 0, nil
}

func (f fakeCategoryBrandStore) Create(ctx context.Context, _ *gorm.DB, relation *do.GoodsCategoryBrandDO) error {
	if f.create != nil {
		return f.create(ctx, relation)
	}
	return nil
}

func (fakeCategoryBrandStore) Update(context.Context, *gorm.DB, *do.GoodsCategoryBrandDO) error {
	return nil
}

func (fakeCategoryBrandStore) Delete(context.Context, uint64) error {
	return nil
}
