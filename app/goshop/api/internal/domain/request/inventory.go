package request

type InventoryDetailURI struct {
	GoodsID int32 `uri:"goods_id" binding:"required,gt=0"`
}

type InventoryOrderDetailURI struct {
	OrderSn string `uri:"order_sn" binding:"required"`
}
