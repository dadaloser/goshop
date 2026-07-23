package v1

import (
	"context"
	proto "goshop/api/goods/v1"
	v1 "goshop/app/goods/srv/internal/data/v1"
	v12 "goshop/app/goods/srv/internal/data_search/v1"
	"goshop/app/goods/srv/internal/domain/do"
	"goshop/app/goods/srv/internal/domain/dto"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"
	"sync"

	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/log"
	"strings"

	"github.com/zeromicro/go-zero/core/mr"
)

type GoodsSrv interface {
	// 商品列表
	List(ctx context.Context, opts metav1.ListMeta, req *proto.GoodsFilterRequest, orderBy []string) (*dto.GoodsDTOList, error)

	// 商品详情
	Get(ctx context.Context, ID uint64) (*dto.GoodsDTO, error)

	// 创建商品
	Create(ctx context.Context, goods *dto.GoodsDTO) error

	// 更新商品
	Update(ctx context.Context, goods *dto.GoodsDTO) error

	// 删除商品
	Delete(ctx context.Context, ID uint64) error

	//批量查询商品
	BatchGet(ctx context.Context, ids []uint64) ([]*dto.GoodsDTO, error)
}

type goodsService struct {
	//工厂
	data v1.DataFactory

	searchData v12.SearchFactory

	testTxn txExecutor
}

func newGoods(srv *service) *goodsService {
	return &goodsService{
		data:       srv.data,
		searchData: srv.dataSearch,
	}
}

func (gs *goodsService) beginTxn() txExecutor {
	if gs != nil && gs.testTxn != nil {
		return gs.testTxn
	}
	return gormTxn{db: gs.data.Begin()}
}

// 遍历树结构
func retrieveIDs(category *do.CategoryDO) []uint64 {
	var ids []uint64
	if category == nil || category.ID == 0 {
		return ids
	}
	ids = append(ids, uint64(category.ID))
	for _, child := range category.SubCategory {
		subids := retrieveIDs(child)
		ids = append(ids, subids...)
	}
	return ids
}

func (gs *goodsService) List(ctx context.Context, opts metav1.ListMeta, req *proto.GoodsFilterRequest, orderBy []string) (*dto.GoodsDTOList, error) {
	if req == nil {
		req = &proto.GoodsFilterRequest{}
	}

	searchReq := v12.GoodsFilterRequest{
		GoodsFilterRequest: req,
	}
	if req.TopCategory > 0 {
		category, err := gs.data.Categories().Get(ctx, uint64(req.TopCategory))
		if err != nil {
			log.Errorf("categoryData.Get err: %v", err)
			return nil, err
		}

		var ids []interface{}
		for _, value := range retrieveIDs(category) {
			ids = append(ids, value)
		}
		searchReq.CategoryIDs = ids
	}

	goodsList, err := gs.searchData.Goods().Search(ctx, &searchReq)
	if err != nil {
		log.Errorf("serachData.Search err: %v", err)
		return nil, err
	}
	if goodsList == nil {
		return &dto.GoodsDTOList{}, nil
	}

	log.Debugf("Search es data: %v", goodsList)

	goodsIDs := []uint64{}
	for _, value := range goodsList.Items {
		goodsIDs = append(goodsIDs, uint64(value.ID))
	}

	//通过id批量查询mysql数据
	goods, err := gs.data.Goods().ListByIDs(ctx, goodsIDs, orderBy)
	if err != nil {
		log.Errorf("data.ListByIDs err: %v", err)
		return nil, err
	}
	var ret dto.GoodsDTOList
	ret.TotalCount = int(goodsList.TotalCount)
	if goods == nil {
		return &ret, nil
	}
	for _, value := range goods.Items {
		if value == nil {
			continue
		}
		ret.Items = append(ret.Items, &dto.GoodsDTO{
			GoodsDO: *value,
		})
	}
	return &ret, nil
}

func (gs *goodsService) Get(ctx context.Context, ID uint64) (*dto.GoodsDTO, error) {
	if ID == 0 {
		return nil, errors.WithCode(code.ErrGoodsNotFound, "goods not found")
	}

	goods, err := gs.data.Goods().Get(ctx, ID)
	if err != nil {
		log.Errorf("data.Get err: %v", err)
		return nil, err
	}
	return &dto.GoodsDTO{
		GoodsDO: *goods,
	}, nil
}

func goodsSearchFromDTO(goods *dto.GoodsDTO) do.GoodsSearchDO {
	return do.GoodsSearchDO{
		ID:             goods.ID,
		CategoryID:     goods.CategoryID,
		BrandsID:       goods.BrandsID,
		OnSale:         goods.OnSale,
		ShipFree:       goods.ShipFree,
		IsNew:          goods.IsNew,
		IsHot:          goods.IsHot,
		Name:           goods.Name,
		ClickNum:       goods.ClickNum,
		SoldNum:        goods.SoldNum,
		FavNum:         goods.FavNum,
		MarketPriceFen: goods.MarketPriceFen,
		GoodsBrief:     goods.GoodsBrief,
		ShopPriceFen:   goods.ShopPriceFen,
		SPUCode:        goods.SPUCode,
		SKUCode:        goods.SKUCode,
	}
}

func (gs *goodsService) Create(ctx context.Context, goods *dto.GoodsDTO) (err error) {
	/*
		数据先写mysql，然后写es
	*/
	if err := validateGoodsForWrite(goods, false); err != nil {
		return err
	}

	_, err = gs.data.Brands().Get(ctx, uint64(goods.BrandsID))
	if err != nil {
		return err
	}

	_, err = gs.data.Categories().Get(ctx, uint64(goods.CategoryID))
	if err != nil {
		return err
	}

	//之前的入es的方案是给gorm添加aftercreate
	//分布式事务， 异构数据库的事务， 基于可靠消息最终一致性
	//比较重的方案： 每次都要发送一个事务消息
	//因此引入外部数据库事务来控制事务
	return withTxnExecutor(ctx, gs.beginTxn(), "create goods", func(txn txExecutor) error {
		if err := gs.data.Goods().CreateInTxn(ctx, txn.DB(), &goods.GoodsDO); err != nil {
			log.Errorf("data.CreateInTxn err: %v", err)
			return err
		}
		event, err := newGoodsSyncEvent(goods)
		if err != nil {
			return err
		}
		return gs.data.Outbox().CreateInTxn(ctx, txn.DB(), event)
	})
}

func (gs *goodsService) Update(ctx context.Context, goods *dto.GoodsDTO) (err error) {
	if err := validateGoodsForWrite(goods, true); err != nil {
		return err
	}

	if _, err := gs.data.Goods().Get(ctx, uint64(goods.ID)); err != nil {
		return err
	}

	_, err = gs.data.Brands().Get(ctx, uint64(goods.BrandsID))
	if err != nil {
		return err
	}

	_, err = gs.data.Categories().Get(ctx, uint64(goods.CategoryID))
	if err != nil {
		return err
	}

	return withTxnExecutor(ctx, gs.beginTxn(), "update goods", func(txn txExecutor) error {
		if err := gs.data.Goods().UpdateInTxn(ctx, txn.DB(), &goods.GoodsDO); err != nil {
			return err
		}
		event, err := newGoodsSyncEvent(goods)
		if err != nil {
			return err
		}
		return gs.data.Outbox().CreateInTxn(ctx, txn.DB(), event)
	})
}

func (gs *goodsService) Delete(ctx context.Context, ID uint64) (err error) {
	if ID == 0 {
		return errors.WithCode(code.ErrGoodsInvalid, "goods id is required")
	}
	if _, err := gs.data.Goods().Get(ctx, ID); err != nil {
		return err
	}

	return withTxnExecutor(ctx, gs.beginTxn(), "delete goods", func(txn txExecutor) error {
		if err := gs.data.Goods().DeleteInTxn(ctx, txn.DB(), ID); err != nil {
			return err
		}
		event, err := newGoodsDeleteEvent(ID)
		if err != nil {
			return err
		}
		return gs.data.Outbox().CreateInTxn(ctx, txn.DB(), event)
	})
}

func (gs *goodsService) BatchGet(ctx context.Context, ids []uint64) ([]*dto.GoodsDTO, error) {
	//go-zero 非常好用， 但是我们自己去做并发的话 - 一次性启动多个goroutine
	var ret []*dto.GoodsDTO
	var callFuncs []func() error
	var mu sync.Mutex //细节:注意并发调用 ,也可使用sync.map来实现
	for _, value := range ids {
		//大坑,一定要引入临时变量,variable在闭包中被引用时会有问题,导致所有的goroutine都使用同一个变量,最终导致结果不正确
		tmp := value
		callFuncs = append(callFuncs, func() error {
			goodsDTO, err := gs.Get(ctx, tmp)
			mu.Lock()
			ret = append(ret, goodsDTO)
			mu.Unlock()
			return err
		})
	}
	err := mr.Finish(callFuncs...)
	if err != nil {
		return nil, err
	}
	//ds, err := gs.data.ListByIDs(ctx, ids, []string{})
	//if err != nil {
	//	return nil, err
	//}
	//for _, value := range ds.Items {
	//	ret = append(ret, &dto.GoodsDTO{
	//		GoodsDO: *value,
	//	})
	//}
	return ret, nil
}

var _ GoodsSrv = &goodsService{}

func validateGoodsForWrite(goods *dto.GoodsDTO, requireID bool) error {
	if goods == nil {
		return errors.WithCode(code.ErrGoodsInvalid, "goods is required")
	}
	if requireID && goods.ID <= 0 {
		return errors.WithCode(code.ErrGoodsInvalid, "goods id is required")
	}

	goods.Name = strings.TrimSpace(goods.Name)
	goods.GoodsSn = strings.TrimSpace(goods.GoodsSn)
	goods.SPUCode = strings.TrimSpace(goods.SPUCode)
	goods.SKUCode = strings.TrimSpace(goods.SKUCode)
	if goods.SPUCode == "" {
		goods.SPUCode = goods.GoodsSn
	}
	if goods.SKUCode == "" {
		goods.SKUCode = goods.GoodsSn
	}
	goods.GoodsBrief = strings.TrimSpace(goods.GoodsBrief)
	goods.GoodsFrontImage = strings.TrimSpace(goods.GoodsFrontImage)

	if goods.CategoryID <= 0 || goods.BrandsID <= 0 {
		return errors.WithCode(code.ErrGoodsInvalid, "category_id and brand_id are required")
	}
	if goods.Name == "" || goods.GoodsSn == "" {
		return errors.WithCode(code.ErrGoodsInvalid, "name and goods_sn are required")
	}
	if goods.MarketPriceFen < 0 || goods.ShopPriceFen < 0 {
		return errors.WithCode(code.ErrGoodsInvalid, "goods price must not be negative")
	}
	return nil
}
