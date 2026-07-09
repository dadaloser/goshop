package mysql

import (
	"context"
	"strings"

	"goshop/app/inventory/srv/internal/domain/do"
	"goshop/app/pkg/code"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"goshop/app/inventory/srv/internal/data/v1"
	"goshop/pkg/log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type inventorys struct {
	db *gorm.DB
}

func inventoryDB(db *gorm.DB, txn *gorm.DB) *gorm.DB {
	if txn != nil {
		return txn
	}
	return db
}

// 更新库存状态
func (i *inventorys) UpdateStockSellDetailStatus(ctx context.Context, txn *gorm.DB, ordersn string, status int32) error {
	ordersn = strings.TrimSpace(ordersn)
	if ordersn == "" {
		return errors.WithCode(code.ErrInvSellDetailNotFound, "inventory sell detail not found")
	}
	if status <= 0 {
		return errors.WithCode(code2.ErrValidation, "inventory sell detail status is invalid")
	}

	db := inventoryDB(i.db, txn)

	//update语句如果没有更新的话那么不会报错，但是他会返回一个影响的行数，
	//所以我们可以根据影响的行数来判断是否更新成功
	result := db.WithContext(ctx).Model(do.StockSellDetailDO{}).Where("order_sn = ?", ordersn).Update("status", status)
	if result.Error != nil {
		return errors.WithCode(code2.ErrDatabase, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.WithCode(code.ErrInvSellDetailNotFound, "inventory sell detail not found")
	}
	return nil
}

func (i *inventorys) GetSellDetail(ctx context.Context, txn *gorm.DB, ordersn string) (*do.StockSellDetailDO, error) {
	ordersn = strings.TrimSpace(ordersn)
	if ordersn == "" {
		return nil, errors.WithCode(code.ErrInvSellDetailNotFound, "inventory sell detail not found")
	}

	db := inventoryDB(i.db, txn)
	var orderSellDetail do.StockSellDetailDO
	err := db.WithContext(ctx).Where("order_sn = ?", ordersn).First(&orderSellDetail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrInvSellDetailNotFound, err.Error())
		}
		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}
	return &orderSellDetail, err
}

func (i *inventorys) Reduce(ctx context.Context, txn *gorm.DB, goodsID uint64, num int) error {
	if goodsID == 0 {
		return errors.WithCode(code.ErrInventoryNotFound, "inventory not found")
	}
	if num <= 0 {
		return errors.WithCode(code2.ErrValidation, "inventory quantity is invalid")
	}

	db := inventoryDB(i.db, txn)
	tx := db.WithContext(ctx).
		Model(&do.InventoryDO{}).
		Where("goods = ?", goodsID).
		Where("available >= ?", num).
		Updates(map[string]interface{}{
			"available": gorm.Expr("available - ?", num),
			"locked":    gorm.Expr("locked + ?", num),
			"stocks":    gorm.Expr("stocks - ?", num),
		})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrInvNotEnough, "库存不足")
	}
	return nil
}

func (i *inventorys) Increase(ctx context.Context, txn *gorm.DB, goodsID uint64, num int) error {
	if goodsID == 0 {
		return errors.WithCode(code.ErrInventoryNotFound, "inventory not found")
	}
	if num <= 0 {
		return errors.WithCode(code2.ErrValidation, "inventory quantity is invalid")
	}

	db := inventoryDB(i.db, txn)
	tx := db.WithContext(ctx).
		Model(&do.InventoryDO{}).
		Where("goods = ?", goodsID).
		Where("locked >= ?", num).
		Updates(map[string]interface{}{
			"available": gorm.Expr("available + ?", num),
			"locked":    gorm.Expr("locked - ?", num),
			"stocks":    gorm.Expr("stocks + ?", num),
		})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrInventoryNotFound, "inventory not found")
	}
	return nil
}

func (i *inventorys) ConfirmSell(ctx context.Context, txn *gorm.DB, goodsID uint64, num int) error {
	if goodsID == 0 {
		return errors.WithCode(code.ErrInventoryNotFound, "inventory not found")
	}
	if num <= 0 {
		return errors.WithCode(code2.ErrValidation, "inventory quantity is invalid")
	}

	db := inventoryDB(i.db, txn)
	tx := db.WithContext(ctx).
		Model(&do.InventoryDO{}).
		Where("goods = ?", goodsID).
		Where("locked >= ?", num).
		Updates(map[string]interface{}{
			"locked": gorm.Expr("locked - ?", num),
			"sold":   gorm.Expr("sold + ?", num),
		})
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.WithCode(code.ErrInventoryNotFound, "inventory not found")
	}
	return nil
}

func (i *inventorys) CreateStockSellDetail(ctx context.Context, txn *gorm.DB, detail *do.StockSellDetailDO) error {
	if err := validateStockSellDetail(detail); err != nil {
		return err
	}

	db := inventoryDB(i.db, txn)

	tx := db.WithContext(ctx).Create(detail)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

func (i *inventorys) CreateStockSellDetailIfAbsent(ctx context.Context, txn *gorm.DB, detail *do.StockSellDetailDO) (bool, error) {
	if err := validateStockSellDetail(detail); err != nil {
		return false, err
	}

	db := inventoryDB(i.db, txn)

	tx := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "order_sn"}},
			DoNothing: true,
		}).
		Create(detail)
	if tx.Error != nil {
		return false, errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return tx.RowsAffected > 0, nil
}

func (i *inventorys) Create(ctx context.Context, inv *do.InventoryDO) error {
	if err := normalizeInventory(inv); err != nil {
		return err
	}

	//设置库存， 如果我要更新库存
	tx := i.db.WithContext(ctx).Create(inv)
	if tx.Error != nil {
		return errors.WithCode(code2.ErrDatabase, tx.Error.Error())
	}
	return nil
}

func (i *inventorys) Get(ctx context.Context, goodsID uint64) (*do.InventoryDO, error) {
	if goodsID == 0 {
		return nil, errors.WithCode(code.ErrInventoryNotFound, "inventory not found")
	}

	inv := do.InventoryDO{}
	err := i.db.WithContext(ctx).Where("goods = ?", goodsID).First(&inv).Error
	if err != nil {
		log.Errorf("get inv err: %v", err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.WithCode(code.ErrInventoryNotFound, err.Error())
		}

		return nil, errors.WithCode(code2.ErrDatabase, err.Error())
	}

	inv.Stocks = inv.Available

	return &inv, nil
}

func normalizeInventory(inv *do.InventoryDO) error {
	if inv == nil || inv.Goods <= 0 {
		return errors.WithCode(code2.ErrValidation, "inventory is invalid")
	}
	if inv.Stocks < 0 || inv.Total < 0 || inv.Available < 0 || inv.Locked < 0 || inv.Sold < 0 {
		return errors.WithCode(code2.ErrValidation, "inventory is invalid")
	}

	lifecycleProvided := inv.Total > 0 || inv.Available > 0 || inv.Locked > 0 || inv.Sold > 0
	if !lifecycleProvided {
		inv.Total = inv.Stocks
		inv.Available = inv.Stocks
		inv.Locked = 0
		inv.Sold = 0
	} else if inv.Total == 0 {
		inv.Total = inv.Available + inv.Locked + inv.Sold
	}

	if inv.Total < inv.Available+inv.Locked+inv.Sold {
		return errors.WithCode(code2.ErrValidation, "inventory is invalid")
	}

	inv.Stocks = inv.Available
	return nil
}

func validateStockSellDetail(detail *do.StockSellDetailDO) error {
	if detail == nil || strings.TrimSpace(detail.OrderSn) == "" {
		return errors.WithCode(code.ErrInvSellDetailNotFound, "inventory sell detail not found")
	}
	if detail.Status <= 0 {
		return errors.WithCode(code2.ErrValidation, "inventory sell detail status is invalid")
	}
	for _, item := range detail.Detail {
		if item.Goods <= 0 || item.Num <= 0 {
			return errors.WithCode(code2.ErrValidation, "inventory sell detail is invalid")
		}
	}
	return nil
}

func newInventorys(data *mysqlStore) *inventorys {
	return &inventorys{db: data.db}
}

var _ v1.InventoryStore = &inventorys{}
