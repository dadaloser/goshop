package v1

import (
	proto "goshop/api/goods/v1"
	proto2 "goshop/api/inventory/v1"

	"gorm.io/gorm"
)

type DataFactory interface {
	Orders() OrderStore
	ShopCarts() ShopCartStore
	//数据来自grpc,如果想要对上层屏蔽,思考一下如何处理才能于proto文件分离
	//重新写一个store接口,在实现时调用grpc接口,这样就能与proto文件分离了
	Goods() proto.GoodsClient
	Inventories() proto2.InventoryClient

	Begin() *gorm.DB
}
