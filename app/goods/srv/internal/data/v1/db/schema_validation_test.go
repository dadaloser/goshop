package db

import "testing"

func TestValidateGoodsSchemaRejectsNilDB(t *testing.T) {
	if err := validateGoodsSchema(nil); err == nil {
		t.Fatal("validateGoodsSchema(nil) error = nil, want error")
	}
}

func TestGoodsSchemaChecksRequireFenColumnsAndForbidLegacyFloatColumns(t *testing.T) {
	var goodsCheck *schemaTableCheck
	for i := range goodsSchemaChecks() {
		check := goodsSchemaChecks()[i]
		if check.model.TableName() == "goods" {
			goodsCheck = &check
			break
		}
	}
	if goodsCheck == nil {
		t.Fatal("goodsSchemaChecks() missing goods table")
	}
	assertContainsAll(t, goodsCheck.required, []string{"market_price_fen", "shop_price_fen", "goods_desc"})
	assertContainsAll(t, goodsCheck.forbidden, []string{"market_price", "shop_price"})
}

func assertContainsAll(t *testing.T, got []string, want []string) {
	t.Helper()
	index := make(map[string]struct{}, len(got))
	for _, value := range got {
		index[value] = struct{}{}
	}
	for _, value := range want {
		if _, ok := index[value]; !ok {
			t.Fatalf("list %v does not contain %q", got, value)
		}
	}
}
