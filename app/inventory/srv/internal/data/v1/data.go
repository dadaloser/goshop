package v1

import "gorm.io/gorm"

type DataFactory interface {
	Inventories() InventoryStore

	Begin() *gorm.DB
}
