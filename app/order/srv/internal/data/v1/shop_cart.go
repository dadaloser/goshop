package v1

import (
	"context"
	"goshop/app/order/srv/internal/domain/do"

	"gorm.io/gorm"

	metav1 "goshop/pkg/common/meta/v1"
)

type ShopCartStore interface {
	List(ctx context.Context, userID uint64, checked bool, meta metav1.ListMeta, orderBy []string) (*do.ShoppingCartDOList, error)
	Create(ctx context.Context, cartItem *do.ShoppingCartDO) error
	Get(ctx context.Context, userID, goodsID uint64) (*do.ShoppingCartDO, error)
	UpdateNum(ctx context.Context, cartItem *do.ShoppingCartDO) error
	Delete(ctx context.Context, userID, ID uint64) error
	ClearCheck(ctx context.Context, userID uint64) error

	DeleteByGoodsIDs(ctx context.Context, txn *gorm.DB, userID uint64, goodsIDs []int32) error
	RestoreCheckedItems(ctx context.Context, txn *gorm.DB, userID uint64, items []*do.OrderGoods) error
}
