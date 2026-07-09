package do

import (
	"database/sql/driver"
	"encoding/json"
	bgorm "goshop/app/pkg/gorm"
)

type GoodsDetail struct {
	Goods int32
	Num   int32
}

type GoodsDetailList []GoodsDetail

func (a GoodsDetailList) Len() int           { return len(a) }
func (a GoodsDetailList) Less(i, j int) bool { return a[i].Goods < a[j].Goods }
func (a GoodsDetailList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (g GoodsDetailList) Value() (driver.Value, error) {
	return json.Marshal(g)
}

// 实现 sql.Scanner 接口，Scan 将 value 扫描至 Jsonb
func (g *GoodsDetailList) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &g)
}

type InventoryDO struct {
	bgorm.BaseModel
	Goods     int32 `gorm:"type:int;index"`
	Stocks    int32 `gorm:"type:int"`
	Total     int32 `gorm:"type:int"`
	Available int32 `gorm:"type:int"`
	Locked    int32 `gorm:"type:int"`
	Sold      int32 `gorm:"type:int"`
	Version   int32 `gorm:"type:int"` //分布式锁的乐观锁
}

func (id *InventoryDO) TableName() string {
	return "inventory"
}

//type InventoryNew struct {
//	bgorm.BaseModel
//	Goods   int32 `gorm:"type:int;index"`
//	Stocks  int32 `gorm:"type:int"`
//	Version int32 `gorm:"type:int"` //分布式锁的乐观锁
//	Freeze  int32 `gorm:"type:int"` //冻结库存
//}

//type Delivery struct {
//	Goods   int32  `gorm:"type:int;index"`
//	Nums    int32  `gorm:"type:int"`
//	OrderSn string `gorm:"type:varchar(200)"`
//	Status  string `gorm:"type:varchar(200)"` //1. 表示等待支付 2. 表示支付成功 3. 失败
//}

type StockSellDetailDO struct {
	OrderSn string          `gorm:"type:varchar(200);index:idx_order_sn,unique;"`
	Status  int32           `gorm:"type:varchar(200)"` //1 表示已锁定 2. 表示已归还 3. 表示已确认
	Detail  GoodsDetailList `gorm:"type:varchar(200)"`
}

func (ssd *StockSellDetailDO) TableName() string {
	return "stockselldetail"
}
