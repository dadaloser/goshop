package service

import (
	"testing"

	"goshop/app/order/srv/internal/boundary"
)

func TestNormalizeGoodsPriceFen(t *testing.T) {
	if got := normalizeGoodsPriceFen(boundary.GoodsInfo{ShopPriceFen: 1234, ShopPrice: 9.99}); got != 1234 {
		t.Fatalf("normalizeGoodsPriceFen() = %d, want 1234", got)
	}
	if got := normalizeGoodsPriceFen(boundary.GoodsInfo{ShopPrice: 12.34}); got != 1234 {
		t.Fatalf("normalizeGoodsPriceFen() legacy = %d, want 1234", got)
	}
}
