package db

import "testing"

func TestValidateOrderSchemaRejectsNilDB(t *testing.T) {
	if err := validateOrderSchema(nil); err == nil {
		t.Fatal("validateOrderSchema(nil) error = nil, want error")
	}
}

func TestOrderSchemaChecksIncludePaymentWorkflowTables(t *testing.T) {
	want := map[string]bool{"order_refund_requests": false, "order_refund_outbox": false, "payment_events": false, "payment_reconciliation_runs": false, "payment_reconciliation_items": false}
	for _, check := range orderSchemaChecks() {
		if _, ok := want[check.model.TableName()]; ok {
			want[check.model.TableName()] = true
		}
	}
	for table, found := range want {
		if !found {
			t.Errorf("payment workflow schema check missing %s", table)
		}
	}
}

func TestOrderSchemaChecksRequireFenColumnsAndForbidLegacyFloatColumns(t *testing.T) {
	var orderInfoCheck *schemaTableCheck
	var orderGoodsCheck *schemaTableCheck
	for i := range orderSchemaChecks() {
		check := orderSchemaChecks()[i]
		switch check.model.TableName() {
		case "orderinfo":
			orderInfoCheck = &check
		case "ordergoods":
			orderGoodsCheck = &check
		}
	}
	if orderInfoCheck == nil || orderGoodsCheck == nil {
		t.Fatalf("order schema checks missing order tables: orderinfo=%v ordergoods=%v", orderInfoCheck != nil, orderGoodsCheck != nil)
	}
	assertContainsAll(t, orderInfoCheck.required, []string{"order_mount_fen"})
	assertContainsAll(t, orderInfoCheck.forbidden, []string{"order_mount"})
	assertContainsAll(t, orderGoodsCheck.required, []string{"goods_price_fen"})
	assertContainsAll(t, orderGoodsCheck.forbidden, []string{"goods_price"})
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
