package do

import (
	bgorm "goshop/app/pkg/gorm"
)

type BannerDO struct {
	bgorm.BaseModel
	Image string `gorm:"type:varchar(200);not null"`
	Url   string `gorm:"type:varchar(200);not null"`
	Index int32  `gorm:"type:int;default:1;not null"`
}

func (BannerDO) TableName() string {
	return "banner"
}

type BannerList struct {
	TotalCount int64       `json:"totalCount,omitempty"`
	Items      []*BannerDO `json:"items"`
}
