package request

type CartItemURI struct {
	ID int32 `uri:"id" binding:"required,gt=0"`
}

type OrderDetailURI struct {
	OrderSn string `uri:"order_sn" binding:"required"`
}

type OrderStatusLogsURI struct {
	OrderSn string `uri:"order_sn" binding:"required"`
}

type OrderListQuery struct {
	Pages       int32 `form:"pages" binding:"omitempty,min=1"`
	PagePerNums int32 `form:"page_per_nums" binding:"omitempty,min=1,max=100"`
}
