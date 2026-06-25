package v1

import (
	"context"
	"goshop/app/inventory/srv/internal/domain/do"

	"gorm.io/gorm"
)

type InventoryStore interface {
	//新建库存信息
	Create(ctx context.Context, inv *do.InventoryDO) error

	//查询商品的库存信息
	Get(ctx context.Context, goodsID uint64) (*do.InventoryDO, error)

	//查询库存销售信息
	GetSellDetail(ctx context.Context, txn *gorm.DB, ordersn string) (*do.StockSellDetailDO, error)

	//扣减库存
	Reduce(ctx context.Context, txn *gorm.DB, goodsID uint64, num int) error

	//新增库存
	Increase(ctx context.Context, txn *gorm.DB, goodsID uint64, num int) error

	//新增库存销售信息
	CreateStockSellDetail(ctx context.Context, txn *gorm.DB, detail *do.StockSellDetailDO) error

	//更新库存销售状态
	UpdateStockSellDetailStatus(ctx context.Context, txn *gorm.DB, ordersn string, status int32) error
}
