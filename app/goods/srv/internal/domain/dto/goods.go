package dto

import "goshop/app/goods/srv/internal/domain/do"

type GoodsDTO struct {
	do.GoodsDO
}

type GoodsDTOList struct {
	TotalCount int         `json:"total_count,omitempty"`
	Items      []*GoodsDTO `json:"data"`
}
