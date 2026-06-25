package mysql

import (
	"context"
	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"goshop/app/inventory/srv/internal/data/v1"
	"goshop/pkg/log"

	"gorm.io/gorm"
)

type inventorys struct {
	db *gorm.DB
}

// 更新库存状态
func (i *inventorys) UpdateStockSellDetailStatus(ctx context.Context, txn *gorm.DB, ordersn string, status int32) error {
	db := i.db
	if txn != nil {
		db = txn
	}

	//update语句如果没有更新的话那么不会报错，但是他会返回一个影响的行数，
	//所以我们可以根据影响的行数来判断是否更新成功
	result := db.Model(do.StockSellDetailDO{}).Where("order_sn = ?", ordersn).Update("status", status)
	if result.Error != nil {
		return errors.WithCode(code2.ErrDatabase, result.Error.Error())
	}

	//这里应该在service层去写代码判断更合理
	//有两种情况都会导致影响的行数为0，一种是没有找到，一种是没有更新
	//if result.RowsAffected == 0 {
	//	return errors.WithCode(code.ErrInvSellDetailNotFound, "inventory sell detail not found")
	//}
	return nil
}

func (i *inventorys) GetSellDetail(ctx context.Context, txn *gorm.DB, ordersn string) (*do.StockSellDetailDO, error) {
	db := i.db
	if txn != nil {
		db = txn
	}
	var orderSellDetail do.StockSellDetailDO
	err := db.Where("order_sn = ?", ordersn).First(&orderSellDetail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrInvSellDetailNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return &orderSellDetail, err
}

func (i *inventorys) Reduce(ctx context.Context, txn *gorm.DB, goodsID uint64, num int) error {
	db := i.db
	if txn != nil {
		db = txn
	}
	return db.Model(&do.InventoryDO{}).Where("goods=?", goodsID).Where("stocks >= ?", num).UpdateColumn("stocks", gorm.Expr("stocks - ?", num)).Error
}

func (i *inventorys) Increase(ctx context.Context, txn *gorm.DB, goodsID uint64, num int) error {
	db := i.db
	if txn != nil {
		db = txn
	}
	err := db.Model(&do.InventoryDO{}).Where("goods=?", goodsID).UpdateColumn("stocks", gorm.Expr("stocks + ?", num)).Error
	return err
}

func (i *inventorys) CreateStockSellDetail(ctx context.Context, txn *gorm.DB, detail *do.StockSellDetailDO) error {
	db := i.db
	if txn != nil {
		db = txn
	}

	tx := db.Create(&detail)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

func (i *inventorys) Create(ctx context.Context, inv *do.InventoryDO) error {
	//设置库存， 如果我要更新库存
	tx := i.db.Create(&inv)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

func (i *inventorys) Get(ctx context.Context, goodsID uint64) (*do.InventoryDO, error) {
	inv := do.InventoryDO{}
	err := i.db.Where("goods = ?", goodsID).First(&inv).Error
	if err != nil {
		log.Errorf("get inv err: %v", err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrInventoryNotFound, err.Error())
		}

		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	return &inv, nil
}

func newInventorys(data *mysqlStore) *inventorys {
	return &inventorys{db: data.db}
}

var _ v1.InventoryStore = &inventorys{}
