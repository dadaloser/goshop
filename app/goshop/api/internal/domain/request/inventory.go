package request

type InventoryDetailURI struct {
	GoodsID int32 `uri:"goods_id" binding:"required,gt=0"`
}
