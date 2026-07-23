package dto

import "goshop/app/order/srv/internal/domain/do"

type OrderDTO struct {
	do.OrderInfoDO
	StatusReason    string
	StatusSource    string
	StatusOperator  string
	RefundAmountFen int64
	ActorUserID     int32
	CorrelationID   string
	RequestID       string
}

type OrderDTOList struct {
	TotalCount int64       `json:"totalCount,omitempty"`
	Items      []*OrderDTO `json:"data"`
}
