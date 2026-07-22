package v1

import (
	proto "goshop/api/goods/v1"
	"testing"
)

func TestCreateGoodsInfoToDTOUsesFenFields(t *testing.T) {
	dto := createGoodsInfoToDTO(&proto.CreateGoodsInfo{
		Id:             1,
		Name:           "goods",
		GoodsSn:        "g-1",
		CategoryId:     2,
		BrandId:        3,
		MarketPriceFen: 1234,
		ShopPriceFen:   899,
	})

	if dto.MarketPriceFen != 1234 || dto.ShopPriceFen != 899 {
		t.Fatalf("money fen fields = (%d,%d), want (1234,899)", dto.MarketPriceFen, dto.ShopPriceFen)
	}
	if dto.MarketPrice != 12.34 || dto.ShopPrice != 8.99 {
		t.Fatalf("legacy float fields = (%v,%v), want (12.34,8.99)", dto.MarketPrice, dto.ShopPrice)
	}
}
