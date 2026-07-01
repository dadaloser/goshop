package v1

import "gorm.io/gorm"

type DataFactory interface {
	Orders() OrderStore
	ShopCarts() ShopCartStore

	Begin() *gorm.DB
}
