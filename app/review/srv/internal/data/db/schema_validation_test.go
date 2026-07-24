package db

import "testing"

func TestValidateReviewSchemaRejectsNilDB(t *testing.T) {
	if err := validateReviewSchema(nil); err == nil {
		t.Fatal("validateReviewSchema(nil) error = nil, want error")
	}
}

func TestReviewSchemaChecksRequireCoreOutboxAndRatings(t *testing.T) {
	var reviewCheck *schemaTableCheck
	var outboxCheck *schemaTableCheck
	var ratingCheck *schemaTableCheck
	for i := range reviewSchemaChecks() {
		check := reviewSchemaChecks()[i]
		switch check.model.TableName() {
		case "reviews":
			reviewCheck = &check
		case "review_outbox_events":
			outboxCheck = &check
		case "review_product_ratings":
			ratingCheck = &check
		}
	}
	if reviewCheck == nil || outboxCheck == nil || ratingCheck == nil {
		t.Fatalf("review schema checks missing required tables: reviews=%v outbox=%v ratings=%v", reviewCheck != nil, outboxCheck != nil, ratingCheck != nil)
	}

	assertContainsAll(t, reviewCheck.required, []string{"user_id", "order_sn", "goods_id", "rating", "status"})
	assertContainsAll(t, outboxCheck.required, []string{"event_key", "goods_id", "status", "next_attempt_at", "completed_at"})
	assertContainsAll(t, ratingCheck.required, []string{"goods_id", "approved_count", "rating_sum", "average_milli", "rebuilt_at"})
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
