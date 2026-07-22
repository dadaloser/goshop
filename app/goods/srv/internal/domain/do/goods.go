package do

import (
	"encoding/json"

	"database/sql/driver"
	gorm2 "goshop/app/pkg/gorm"
	"goshop/pkg/money"
)

type GoodsSearchDO struct {
	ID         int32 `json:"id"`
	CategoryID int32 `json:"category_id"`
	BrandsID   int32 `json:"brands_id"`
	OnSale     bool  `json:"on_sale"`
	ShipFree   bool  `json:"ship_free"`
	IsNew      bool  `json:"is_new"`
	IsHot      bool  `json:"is_hot"`

	Name           string  `json:"name"`
	ClickNum       int32   `json:"click_num"`
	SoldNum        int32   `json:"sold_num"`
	FavNum         int32   `json:"fav_num"`
	MarketPrice    float32 `json:"market_price"`
	MarketPriceFen int64   `json:"market_price_fen"`
	GoodsBrief     string  `json:"goods_brief"`
	ShopPrice      float32 `json:"shop_price"`
	ShopPriceFen   int64   `json:"shop_price_fen"`
}

func (GoodsSearchDO) GetIndexName() string {
	return "goods"
}

type GoodsSearchDOList struct {
	TotalCount int64            `json:"totalCount,omitempty"`
	Items      []*GoodsSearchDO `json:"items"`
}

type GoodsDO struct {
	gorm2.BaseModel

	CategoryID int32      `gorm:"type:int;not null"`
	Category   CategoryDO `gorm:"foreignKey:CategoryID;references:ID" json:"category"`
	BrandsID   int32      `gorm:"type:int;not null"`
	Brands     BrandsDO   `gorm:"foreignKey:BrandsID;references:ID" json:"Brands"`

	OnSale   bool `gorm:"default:false;not null"`
	ShipFree bool `gorm:"default:false;not null"`
	IsNew    bool `gorm:"default:false;not null"`
	IsHot    bool `gorm:"default:false;not null"`

	Name            string   `gorm:"type:varchar(50);not null"`
	GoodsSn         string   `gorm:"type:varchar(50);not null"`
	ClickNum        int32    `gorm:"type:int;default:0;not null"`
	SoldNum         int32    `gorm:"type:int;default:0;not null"`
	FavNum          int32    `gorm:"type:int;default:0;not null"`
	MarketPrice     float32  `gorm:"not null"`
	MarketPriceFen  int64    `gorm:"type:bigint;not null;default:0"`
	ShopPrice       float32  `gorm:"not null"`
	ShopPriceFen    int64    `gorm:"type:bigint;not null;default:0"`
	GoodsBrief      string   `gorm:"type:varchar(100);not null"`
	Images          GormList `gorm:"type:varchar(1000);not null"`
	DescImages      GormList `gorm:"type:varchar(1000);not null"`
	GoodsFrontImage string   `gorm:"type:varchar(200);not null"`
}

func (GoodsDO) TableName() string {
	return "goods"
}

func (g GoodsDO) EffectiveMarketPriceFen() int64 {
	if g.MarketPriceFen != 0 || g.MarketPrice == 0 {
		return g.MarketPriceFen
	}
	return money.FromLegacyFloat32Yuan(g.MarketPrice).Int64()
}

func (g GoodsDO) EffectiveShopPriceFen() int64 {
	if g.ShopPriceFen != 0 || g.ShopPrice == 0 {
		return g.ShopPriceFen
	}
	return money.FromLegacyFloat32Yuan(g.ShopPrice).Int64()
}

func (g *GoodsDO) SyncLegacyMoneyFields() {
	if g == nil {
		return
	}
	g.MarketPriceFen = g.EffectiveMarketPriceFen()
	g.ShopPriceFen = g.EffectiveShopPriceFen()
	g.MarketPrice = money.NewFen(g.MarketPriceFen).Float32Yuan()
	g.ShopPrice = money.NewFen(g.ShopPriceFen).Float32Yuan()
}

func (g GoodsSearchDO) EffectiveMarketPriceFen() int64 {
	if g.MarketPriceFen != 0 || g.MarketPrice == 0 {
		return g.MarketPriceFen
	}
	return money.FromLegacyFloat32Yuan(g.MarketPrice).Int64()
}

func (g GoodsSearchDO) EffectiveShopPriceFen() int64 {
	if g.ShopPriceFen != 0 || g.ShopPrice == 0 {
		return g.ShopPriceFen
	}
	return money.FromLegacyFloat32Yuan(g.ShopPrice).Int64()
}

func (g *GoodsSearchDO) SyncLegacyMoneyFields() {
	if g == nil {
		return
	}
	g.MarketPriceFen = g.EffectiveMarketPriceFen()
	g.ShopPriceFen = g.EffectiveShopPriceFen()
	g.MarketPrice = money.NewFen(g.MarketPriceFen).Float32Yuan()
	g.ShopPrice = money.NewFen(g.ShopPriceFen).Float32Yuan()
}

// 去掉gorm的依赖
type GormList []string

func (g GormList) Value() (driver.Value, error) {
	return json.Marshal(g)
}

// 实现 sql.Scanner 接口，Scan 将 value 扫描至 Jsonb
func (g *GormList) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &g)
}

type GoodsDOList struct {
	TotalCount int64      `json:"totalCount,omitempty"`
	Items      []*GoodsDO `json:"items"`
}
