package db

import (
	"context"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"

	v1 "goshop/app/goods/srv/internal/data/v1"
	"goshop/app/goods/srv/internal/domain/do"

	"gorm.io/gorm"
)

type goods struct {
	db *gorm.DB
}

func (g *goods) Begin() *gorm.DB {
	return g.db.Begin()
}

func newGoods(factory *mysqlFactory) *goods {
	goods := &goods{
		db: factory.db,
	}
	return goods
}

func (g *goods) CreateInTxn(ctx context.Context, txn *gorm.DB, goods *do.GoodsDO) error {
	if txn == nil || goods == nil {
		return errors.WithCode(code.ErrGoodsInvalid, "goods is required")
	}

	tx := txn.WithContext(ctx).Create(goods)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

func (g *goods) UpdateInTxn(ctx context.Context, txn *gorm.DB, goods *do.GoodsDO) error {
	if txn == nil || goods == nil || goods.ID <= 0 {
		return errors.WithCode(code.ErrGoodsNotFound, "goods not found")
	}

	tx := txn.WithContext(ctx).Model(&do.GoodsDO{}).
		Where("id = ?", goods.ID).
		Updates(goodsUpdateValues(goods))
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		if _, err := g.Get(ctx, uint64(goods.ID)); err != nil {
			return err
		}
	}
	return nil
}

func (g *goods) DeleteInTxn(ctx context.Context, txn *gorm.DB, ID uint64) error {
	if txn == nil || ID == 0 {
		return errors.WithCode(code.ErrGoodsNotFound, "goods not found")
	}

	tx := txn.WithContext(ctx).Where("id = ?", ID).Delete(&do.GoodsDO{})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrGoodsNotFound, "goods not found")
	}
	return nil
}

//func NewGoods(db *gorm.DB) *goods {
//	return &goods{
//		db: db,
//	}
//}

func (g *goods) List(ctx context.Context, orderBy []string, opts metav1.ListMeta) (*do.GoodsDOList, error) {
	//实现gorm查询
	ret := &do.GoodsDOList{}

	//分页
	var limit, offset int
	if opts.PageSize == 0 {
		limit = 10
	} else {
		limit = opts.PageSize
	}

	if opts.Page > 0 {
		offset = (opts.Page - 1) * limit
	}

	//排序
	query := g.db.WithContext(ctx).Preload("Category").Preload("Brands")
	for _, value := range orderBy {
		//坑
		query = query.Order(value)
	}

	d := query.Offset(offset).Limit(limit).Find(&ret.Items).Count(&ret.TotalCount)
	if d.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, d.Error.Error())
	}
	return ret, nil
}

func (g *goods) Get(ctx context.Context, ID uint64) (*do.GoodsDO, error) {
	good := &do.GoodsDO{}
	err := g.db.WithContext(ctx).Preload("Category").Preload("Brands").First(good, ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrGoodsNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return good, nil
}

func (g *goods) ListByIDs(ctx context.Context, ids []uint64, orderBy []string) (*do.GoodsDOList, error) {
	//实现gorm查询
	ret := &do.GoodsDOList{}

	//排序
	query := g.db.WithContext(ctx).Preload("Category").Preload("Brands")
	for _, value := range orderBy {
		query = query.Order(value)
	}

	d := query.Where("id in ?", ids).Find(&ret.Items).Count(&ret.TotalCount)
	if d.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, d.Error.Error())
	}
	return ret, nil
}

func (g *goods) Create(ctx context.Context, goods *do.GoodsDO) error {
	if goods == nil {
		return errors.WithCode(code.ErrGoodsInvalid, "goods is required")
	}

	tx := g.db.WithContext(ctx).Create(goods)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

func (g *goods) Update(ctx context.Context, goods *do.GoodsDO) error {
	if goods == nil || goods.ID <= 0 {
		return errors.WithCode(code.ErrGoodsNotFound, "goods not found")
	}

	tx := g.db.WithContext(ctx).Model(&do.GoodsDO{}).
		Where("id = ?", goods.ID).
		Updates(goodsUpdateValues(goods))
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrGoodsNotFound, "goods not found")
	}
	return nil
}

func (g *goods) Delete(ctx context.Context, ID uint64) error {
	if ID == 0 {
		return errors.WithCode(code.ErrGoodsNotFound, "goods not found")
	}

	tx := g.db.WithContext(ctx).Where("id = ?", ID).Delete(&do.GoodsDO{})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrGoodsNotFound, "goods not found")
	}
	return nil
}

var _ v1.GoodsStore = &goods{}

func goodsUpdateValues(goods *do.GoodsDO) map[string]interface{} {
	return map[string]interface{}{
		"category_id":       goods.CategoryID,
		"brands_id":         goods.BrandsID,
		"on_sale":           goods.OnSale,
		"ship_free":         goods.ShipFree,
		"is_new":            goods.IsNew,
		"is_hot":            goods.IsHot,
		"name":              goods.Name,
		"goods_sn":          goods.GoodsSn,
		"click_num":         goods.ClickNum,
		"sold_num":          goods.SoldNum,
		"fav_num":           goods.FavNum,
		"market_price":      goods.MarketPrice,
		"shop_price":        goods.ShopPrice,
		"goods_brief":       goods.GoodsBrief,
		"images":            goods.Images,
		"desc_images":       goods.DescImages,
		"goods_front_image": goods.GoodsFrontImage,
	}
}
