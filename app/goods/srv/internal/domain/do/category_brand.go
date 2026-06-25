package do

import (
	bgorm "goshop/app/pkg/gorm"
)

type GoodsCategoryBrandDO struct {
	bgorm.BaseModel

	CategoryID int32      `gorm:"type:int;index:idx_category_brand,unique"`
	Category   CategoryDO `gorm:"foreignKey:CategoryID;references:ID" json:"category"`

	BrandsID int32    `gorm:"type:int;index:idx_category_brand,unique"`
	Brands   BrandsDO `gorm:"foreignKey:BrandsID;references:ID" json:"brands"`
}

func (GoodsCategoryBrandDO) TableName() string {
	return "goodscategorybrand"
}

type GoodsCategoryBrandList struct {
	TotalCount int64                   `json:"totalCount,omitempty"`
	Items      []*GoodsCategoryBrandDO `json:"items"`
}
