package data

import (
	gpb "goshop/api/goods/v1"
	opb "goshop/api/order/v1"
)

type DataFactory interface {
	Goods() gpb.GoodsClient
	Orders() opb.OrderClient
	Users() UserData
}
