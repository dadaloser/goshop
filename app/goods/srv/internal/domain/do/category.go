package do

import (
	"goshop/app/pkg/gorm"
)

type CategoryDO struct {
	gorm.BaseModel
	Name             string        `gorm:"type:varchar(20);not null" json:"name"`
	ParentCategoryID int32         `json:"parent"`
	ParentCategory   *CategoryDO   `json:"-"`
	SubCategory      []*CategoryDO `gorm:"foreignKey:ParentCategoryID;references:ID" json:"sub_category"`
	Level            int32         `gorm:"type:int;not null;default:1" json:"level"`
	IsTab            bool          `gorm:"default:false;not null" json:"is_tab"`
}

func (CategoryDO) TableName() string {
	return "category"
}

type CategoryDOList struct {
	TotalCount int64         `json:"totalCount,omitempty"`
	Items      []*CategoryDO `json:"items"`
}
