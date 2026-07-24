package mysql

import "testing"

func TestValidateInventorySchemaRejectsNilDB(t *testing.T) {
	if err := validateInventorySchema(nil); err == nil {
		t.Fatal("validateInventorySchema(nil) error = nil, want error")
	}
}

func TestInventorySchemaChecksRequireLifecycleAndAuditTables(t *testing.T) {
	var inventoryCheck *schemaTableCheck
	var detailCheck *schemaTableCheck
	var auditCheck *schemaTableCheck
	for i := range inventorySchemaChecks() {
		check := inventorySchemaChecks()[i]
		switch check.model.TableName() {
		case "inventory":
			inventoryCheck = &check
		case "stockselldetail":
			detailCheck = &check
		case "inventory_adjustment_logs":
			auditCheck = &check
		}
	}
	if inventoryCheck == nil || detailCheck == nil || auditCheck == nil {
		t.Fatalf("inventory schema checks missing required tables: inventory=%v stockselldetail=%v inventory_adjustment_logs=%v", inventoryCheck != nil, detailCheck != nil, auditCheck != nil)
	}

	assertContainsAll(t, inventoryCheck.required, []string{"goods", "stocks", "total", "available", "locked", "sold", "version"})
	assertContainsAll(t, detailCheck.required, []string{"order_sn", "status", "detail"})
	assertContainsAll(t, auditCheck.required, []string{"goods_id", "before_available", "after_available", "actor_user_id", "correlation_id"})
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
