package db

import (
	"context"

	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	v1 "goshop/app/order/srv/internal/data/v1"
	"goshop/app/order/srv/internal/domain/do"
	metav1 "goshop/pkg/common/meta/v1"

	"gorm.io/gorm"
)

type shopCarts struct {
	db *gorm.DB
}

func newShopCarts(factory *dataFactory) *shopCarts {
	return &shopCarts{
		db: factory.db,
	}
}

// 这个在事务中执行，建议大家使用消息队列来实现
func (sc *shopCarts) DeleteByGoodsIDs(ctx context.Context, txn *gorm.DB, userID uint64, goodsIDs []int32) error {
	db := sc.db
	if txn != nil {
		db = txn
	}
	return db.WithContext(ctx).Where("user = ? AND goods IN (?)", userID, goodsIDs).Delete(&do.ShoppingCartDO{}).Error
}

func (sc *shopCarts) RestoreCheckedItems(ctx context.Context, txn *gorm.DB, userID uint64, items []*do.OrderGoods) error {
	db := sc.db
	if txn != nil {
		db = txn
	}

	for _, item := range items {
		if item == nil || item.Goods <= 0 || item.Nums <= 0 {
			continue
		}

		tx := db.WithContext(ctx).Model(&do.ShoppingCartDO{}).
			Where("user = ? AND goods = ?", userID, item.Goods).
			Updates(map[string]interface{}{
				"nums":    gorm.Expr("GREATEST(nums, ?)", item.Nums),
				"checked": true,
			})
		if tx.Error != nil {
			return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
		}
		if tx.RowsAffected > 0 {
			continue
		}

		err := db.WithContext(ctx).Create(&do.ShoppingCartDO{
			User:    int32(userID),
			Goods:   item.Goods,
			Nums:    item.Nums,
			Checked: true,
		}).Error
		if err != nil {
			return errors.WithCode(code2.ErrDatabase, err.Error())
		}
	}
	return nil
}

func (sc *shopCarts) List(ctx context.Context, userID uint64, checked bool, meta metav1.ListMeta, orderBy []string) (*do.ShoppingCartDOList, error) {
	ret := &do.ShoppingCartDOList{}
	countQuery := sc.db.WithContext(ctx).Model(&do.ShoppingCartDO{})
	//分页
	var limit, offset int
	if meta.PageSize == 0 {
		limit = 10
	} else {
		limit = meta.PageSize
	}

	if meta.Page > 0 {
		offset = (meta.Page - 1) * limit
	}

	if userID > 0 {
		countQuery = countQuery.Where("user = ?", userID)
	}
	if checked {
		countQuery = countQuery.Where("checked = ?", true)
	}
	if err := countQuery.Count(&ret.TotalCount).Error; err != nil {
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	//排序
	query := sc.db.WithContext(ctx).Model(&do.ShoppingCartDO{})
	if userID > 0 {
		query = query.Where("user = ?", userID)
	}
	if checked {
		query = query.Where("checked = ?", true)
	}
	for _, value := range orderBy {
		query = query.Order(value)
	}

	d := query.Offset(offset).Limit(limit).Find(&ret.Items)
	if d.Error != nil {
		return nil, errors.WithCode(code2.ErrDatabase, d.Error.Error())
	}
	return ret, nil
}

func (sc *shopCarts) Create(ctx context.Context, cartItem *do.ShoppingCartDO) error {
	tx := sc.db.WithContext(ctx).Create(cartItem)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

func (sc *shopCarts) Get(ctx context.Context, userID, goodsID uint64) (*do.ShoppingCartDO, error) {
	var shopCart do.ShoppingCartDO
	err := sc.db.WithContext(ctx).Where("user = ? AND goods = ?", userID, goodsID).First(&shopCart).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrShopCartItemNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return &shopCart, nil
}

func (sc *shopCarts) UpdateNum(ctx context.Context, cartItem *do.ShoppingCartDO) error {
	tx := sc.db.WithContext(ctx).Model(&do.ShoppingCartDO{}).
		Where("user = ? AND goods = ?", cartItem.User, cartItem.Goods).
		Update("nums", cartItem.Nums).
		Update("checked", cartItem.Checked)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrShopCartItemNotFound, "shop cart item not found")
	}
	return nil
}

func (sc *shopCarts) Delete(ctx context.Context, ID uint64) error {
	tx := sc.db.WithContext(ctx).Where("id = ?", ID).Delete(&do.ShoppingCartDO{})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrShopCartItemNotFound, "shop cart item not found")
	}
	return nil
}

// 清空check状态
func (sc *shopCarts) ClearCheck(ctx context.Context, userID uint64) error {
	tx := sc.db.WithContext(ctx).Model(&do.ShoppingCartDO{}).
		Where("user = ? AND checked = ?", userID, true).
		Update("checked", false)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

// todo : 删除选中商品的购物车记录， 下订单了
// 从架构上来讲，这种实现有两种方案
// 下单后， 直接执行删除购物车的记录，比较简单
// 下单后什么都不做，直接给rocketmq发送一个消息，然后由rocketmq来执行删除购物车的记录(推荐)
var _ v1.ShopCartStore = &shopCarts{}
