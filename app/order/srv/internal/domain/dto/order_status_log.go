package dto

import "goshop/app/order/srv/internal/domain/do"

type OrderStatusLogDTO struct {
	do.OrderStatusLogDO
}

type OrderStatusLogDTOList struct {
	TotalCount int64                `json:"totalCount,omitempty"`
	Items      []*OrderStatusLogDTO `json:"data"`
}
