package db

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRefundFailureDecisionRetriesThenDeadLetters(t *testing.T) {
	now := time.Unix(100, 0)
	retry := refundFailureDecision(2, 3, now)
	if retry.dead || !retry.availableAt.Equal(now.Add(4*time.Second)) {
		t.Fatalf("retry decision=%+v", retry)
	}
	dead := refundFailureDecision(3, 3, now)
	if !dead.dead {
		t.Fatalf("dead decision=%+v", dead)
	}
}

func TestRefundOutboxMigrationEnforcesIdempotencyAndClaimIndex(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "../../../../../../../migrations/202607240001_payment_refund_outbox_reconciliation.up.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sql := string(content)
	for _, fragment := range []string{"UNIQUE KEY `uk_refund_outbox_request`", "KEY `idx_refund_outbox_claim`", "FOREIGN KEY (`refund_request_id`)"} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
}
