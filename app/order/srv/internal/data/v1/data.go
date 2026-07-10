package v1

import "gorm.io/gorm"

type DataFactory interface {
	Orders() OrderStore
	OrderStatusLogs() OrderStatusLogStore
	ShopCarts() ShopCartStore

	Begin() *gorm.DB
}
