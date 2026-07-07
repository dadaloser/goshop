package code

const (
	// ErrShopCartItemNotFound - 404: ShopCart item not found.
	ErrShopCartItemNotFound int = iota + 100701

	// ErrSubmitOrder - 400: Submit order error.
	ErrSubmitOrder

	// ErrNoGoodsSelect - 400: No Goods selected.
	ErrNoGoodsSelect

	// ErrOrderNotFound - 404: Order not found.
	ErrOrderNotFound

	// ErrOrderConflict - 400: Order already exists with different data.
	ErrOrderConflict
)
