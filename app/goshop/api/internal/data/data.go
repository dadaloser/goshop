package data

import (
	gpb "goshop/api/goods/v1"
)

type DataFactory interface {
	Goods() gpb.GoodsClient
	Users() UserData
}
