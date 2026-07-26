package v1

import "testing"

func TestSessionTableNames(t *testing.T) {
	if got := (&UserSessionDO{}).TableName(); got != "user_sessions" {
		t.Fatalf("UserSessionDO.TableName() = %q, want %q", got, "user_sessions")
	}
	if got := (&VerificationCodeDO{}).TableName(); got != "verification_codes" {
		t.Fatalf("VerificationCodeDO.TableName() = %q, want %q", got, "verification_codes")
	}
}
