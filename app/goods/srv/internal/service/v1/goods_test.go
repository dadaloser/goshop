package v1

import (
	"context"
	"testing"

	proto "goshop/api/goods/v1"
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
				goods.ShopPriceFen = -1
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

func TestGetRejectsMissingGoodsID(t *testing.T) {
	svc := &goodsService{}

	_, err := svc.Get(context.Background(), 0)
	if !errors.IsCode(err, code.ErrGoodsNotFound) {
		t.Fatalf("Get() error = %v, want code %d", err, code.ErrGoodsNotFound)
	}
}

func TestListAcceptsNilFilter(t *testing.T) {
	var gotSearchReq *searchv1.GoodsFilterRequest
	svc := &goodsService{
		data: fakeGoodsDataFactory{
			goods: fakeGoodsStore{
				listByIDs: func(context.Context, []uint64, []string) (*do.GoodsDOList, error) {
					return &do.GoodsDOList{}, nil
				},
			},
		},
		searchData: fakeSearchFactory{
			goods: fakeSearchGoodsStore{
				search: func(_ context.Context, req *searchv1.GoodsFilterRequest) (*do.GoodsSearchDOList, error) {
					gotSearchReq = req
					return &do.GoodsSearchDOList{}, nil
				},
			},
		},
	}

	got, err := svc.List(context.Background(), metav1.ListMeta{}, nil, nil)
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if got == nil || got.TotalCount != 0 || len(got.Items) != 0 {
		t.Fatalf("List() = %+v, want empty result", got)
	}
	if gotSearchReq == nil || gotSearchReq.GoodsFilterRequest == nil {
		t.Fatalf("List() search request = %+v, want non-nil embedded filter", gotSearchReq)
	}
}

func TestListTreatsNilStoresAsEmptyResult(t *testing.T) {
	tests := []struct {
		name string
		svc  *goodsService
	}{
		{
			name: "nil search result",
			svc: &goodsService{
				searchData: fakeSearchFactory{
					goods: fakeSearchGoodsStore{
						search: func(context.Context, *searchv1.GoodsFilterRequest) (*do.GoodsSearchDOList, error) {
							return nil, nil
						},
					},
				},
			},
		},
		{
			name: "nil mysql result",
			svc: &goodsService{
				data: fakeGoodsDataFactory{
					goods: fakeGoodsStore{
						listByIDs: func(context.Context, []uint64, []string) (*do.GoodsDOList, error) {
							return nil, nil
						},
					},
				},
				searchData: fakeSearchFactory{
					goods: fakeSearchGoodsStore{
						search: func(context.Context, *searchv1.GoodsFilterRequest) (*do.GoodsSearchDOList, error) {
							return &do.GoodsSearchDOList{
								TotalCount: 1,
								Items:      []*do.GoodsSearchDO{{ID: 1}},
							}, nil
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.svc.List(context.Background(), metav1.ListMeta{}, &proto.GoodsFilterRequest{}, nil)
			if err != nil {
				t.Fatalf("List() error = %v, want nil", err)
			}
			if got == nil || len(got.Items) != 0 {
				t.Fatalf("List() = %+v, want empty items", got)
			}
		})
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

func TestCreateWritesGoodsSyncOutboxEvent(t *testing.T) {
	var createdEvent *do.OutboxEventDO
	svc := &goodsService{
		testTxn: fakeTxn{},
		data: fakeGoodsDataFactory{
			goods: fakeGoodsStore{},
			outbox: fakeOutboxStore{
				createInTxn: func(_ context.Context, _ *gorm.DB, event *do.OutboxEventDO) error {
					createdEvent = event
					return nil
				},
			},
			categories: fakeCategoryStore{},
			brands:     fakeBrandStore{},
		},
	}

	goods := validGoodsDTO()
	goods.ID = 12
	if err := svc.Create(context.Background(), goods); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if createdEvent == nil {
		t.Fatal("Create() did not enqueue outbox event")
	}
	if createdEvent.Action != do.OutboxActionUpsert {
		t.Fatalf("Create() action = %q, want %q", createdEvent.Action, do.OutboxActionUpsert)
	}
	if createdEvent.Topic != do.OutboxTopicGoodsSync {
		t.Fatalf("Create() topic = %q, want %q", createdEvent.Topic, do.OutboxTopicGoodsSync)
	}
}

func TestDeleteWritesGoodsDeleteOutboxEvent(t *testing.T) {
	var createdEvent *do.OutboxEventDO
	svc := &goodsService{
		testTxn: fakeTxn{},
		data: fakeGoodsDataFactory{
			goods: fakeGoodsStore{},
			outbox: fakeOutboxStore{
				createInTxn: func(_ context.Context, _ *gorm.DB, event *do.OutboxEventDO) error {
					createdEvent = event
					return nil
				},
			},
		},
	}

	if err := svc.Delete(context.Background(), 9); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if createdEvent == nil {
		t.Fatal("Delete() did not enqueue outbox event")
	}
	if createdEvent.Action != do.OutboxActionDelete {
		t.Fatalf("Delete() action = %q, want %q", createdEvent.Action, do.OutboxActionDelete)
	}
}

func validGoodsDTO() *dto.GoodsDTO {
	return &dto.GoodsDTO{
		GoodsDO: do.GoodsDO{
			CategoryID:      1,
			BrandsID:        1,
			Name:            "goods",
			GoodsSn:         "goods-sn",
			MarketPriceFen:  1000,
			ShopPriceFen:    800,
			GoodsBrief:      "brief",
			GoodsFrontImage: "front.jpg",
		},
	}
}

type fakeGoodsDataFactory struct {
	goods          datav1.GoodsStore
	outbox         datav1.OutboxStore
	categories     datav1.CategoryStore
	brands         datav1.BrandsStore
	banners        datav1.BannerStore
	categoryBrands datav1.GoodsCategoryBrandStore
}

func (f fakeGoodsDataFactory) Goods() datav1.GoodsStore {
	return f.goods
}

func (f fakeGoodsDataFactory) Outbox() datav1.OutboxStore {
	return f.outbox
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
	get             func(context.Context, uint64) (*do.GoodsDO, error)
	countByCategory func(context.Context, uint64) (int64, error)
	countByBrand    func(context.Context, uint64) (int64, error)
	listByIDs       func(context.Context, []uint64, []string) (*do.GoodsDOList, error)
}

func (f fakeGoodsStore) Get(ctx context.Context, id uint64) (*do.GoodsDO, error) {
	if f.get != nil {
		return f.get(ctx, id)
	}
	return &do.GoodsDO{}, nil
}

func (f fakeGoodsStore) CountByCategory(ctx context.Context, categoryID uint64) (int64, error) {
	if f.countByCategory != nil {
		return f.countByCategory(ctx, categoryID)
	}
	return 0, nil
}

func (f fakeGoodsStore) CountByBrand(ctx context.Context, brandID uint64) (int64, error) {
	if f.countByBrand != nil {
		return f.countByBrand(ctx, brandID)
	}
	return 0, nil
}

func (f fakeGoodsStore) ListByIDs(ctx context.Context, ids []uint64, orderBy []string) (*do.GoodsDOList, error) {
	if f.listByIDs != nil {
		return f.listByIDs(ctx, ids, orderBy)
	}
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
	get    func(context.Context, uint64) (*do.BrandsDO, error)
	delete func(context.Context, uint64) error
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

func (f fakeBrandStore) Delete(ctx context.Context, id uint64) error {
	if f.delete != nil {
		return f.delete(ctx, id)
	}
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

type fakeOutboxStore struct {
	createInTxn  func(context.Context, *gorm.DB, *do.OutboxEventDO) error
	claim        func(context.Context, string, int, int64) ([]*do.OutboxEventDO, error)
	listByStatus func(context.Context, string, string, int) ([]*do.OutboxEventDO, error)
	markDone     func(context.Context, int32) error
	markRetry    func(context.Context, int32, int32, int64, string) error
	markDead     func(context.Context, int32, int32, string) error
	release      func(context.Context, int32) error
}

type fakeTxn struct{}

func (fakeTxn) DB() *gorm.DB {
	return &gorm.DB{}
}

func (fakeTxn) Commit() error {
	return nil
}

func (fakeTxn) Rollback() error {
	return nil
}

func (f fakeOutboxStore) CreateInTxn(ctx context.Context, txn *gorm.DB, event *do.OutboxEventDO) error {
	if f.createInTxn != nil {
		return f.createInTxn(ctx, txn, event)
	}
	return nil
}

func (f fakeOutboxStore) ClaimPending(ctx context.Context, topic string, limit int, nowUnix int64) ([]*do.OutboxEventDO, error) {
	if f.claim != nil {
		return f.claim(ctx, topic, limit, nowUnix)
	}
	return nil, nil
}

func (f fakeOutboxStore) ListByStatus(ctx context.Context, topic, status string, limit int) ([]*do.OutboxEventDO, error) {
	if f.listByStatus != nil {
		return f.listByStatus(ctx, topic, status, limit)
	}
	return nil, nil
}

func (f fakeOutboxStore) MarkDone(ctx context.Context, id int32) error {
	if f.markDone != nil {
		return f.markDone(ctx, id)
	}
	return nil
}

func (f fakeOutboxStore) MarkRetry(ctx context.Context, id int32, retryCount int32, nextAttemptAt int64, lastError string) error {
	if f.markRetry != nil {
		return f.markRetry(ctx, id, retryCount, nextAttemptAt, lastError)
	}
	return nil
}

func (f fakeOutboxStore) MarkDead(ctx context.Context, id int32, retryCount int32, lastError string) error {
	if f.markDead != nil {
		return f.markDead(ctx, id, retryCount, lastError)
	}
	return nil
}

func (f fakeOutboxStore) ReleaseClaim(ctx context.Context, id int32) error {
	if f.release != nil {
		return f.release(ctx, id)
	}
	return nil
}

type fakeSearchGoodsStore struct {
	search func(context.Context, *searchv1.GoodsFilterRequest) (*do.GoodsSearchDOList, error)
	update func(context.Context, *do.GoodsSearchDO) error
	delete func(context.Context, uint64) error
}

func (fakeSearchGoodsStore) Create(context.Context, *do.GoodsSearchDO) error {
	return nil
}

func (f fakeSearchGoodsStore) Delete(ctx context.Context, id uint64) error {
	if f.delete != nil {
		return f.delete(ctx, id)
	}
	return nil
}

func (f fakeSearchGoodsStore) Update(ctx context.Context, goods *do.GoodsSearchDO) error {
	if f.update != nil {
		return f.update(ctx, goods)
	}
	return nil
}

func (f fakeSearchGoodsStore) Search(ctx context.Context, req *searchv1.GoodsFilterRequest) (*do.GoodsSearchDOList, error) {
	if f.search != nil {
		return f.search(ctx, req)
	}
	return nil, nil
}

var _ datav1.DataFactory = fakeGoodsDataFactory{}
var _ datav1.GoodsStore = fakeGoodsStore{}
var _ datav1.OutboxStore = fakeOutboxStore{}
var _ datav1.BrandsStore = fakeBrandStore{}
var _ searchv1.SearchFactory = fakeSearchFactory{}
var _ searchv1.GoodsStore = fakeSearchGoodsStore{}
