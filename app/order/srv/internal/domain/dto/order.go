package dto

import "goshop/app/order/srv/internal/domain/do"

type OrderDTO struct {
	do.OrderInfoDO
	StatusReason   string
	StatusSource   string
	StatusOperator string
}

type OrderDTOList struct {
	TotalCount int64       `json:"totalCount,omitempty"`
	Items      []*OrderDTO `json:"data"`
}
