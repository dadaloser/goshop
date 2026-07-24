package db

import "testing"

func TestValidateUserSchemaRejectsNilDB(t *testing.T) {
	if err := validateUserSchema(nil); err == nil {
		t.Fatal("validateUserSchema(nil) error = nil, want error")
	}
}

func TestUserSchemaChecksRequireIdentitySessionsAndRBAC(t *testing.T) {
	var userCheck *schemaTableCheck
	var sessionCheck *schemaTableCheck
	var scopeCheck *schemaTableCheck
	for i := range userSchemaChecks() {
		check := userSchemaChecks()[i]
		switch check.model.TableName() {
		case "user":
			userCheck = &check
		case "user_sessions":
			sessionCheck = &check
		case "user_resource_scopes":
			scopeCheck = &check
		}
	}
	if userCheck == nil || sessionCheck == nil || scopeCheck == nil {
		t.Fatalf("user schema checks missing required tables: user=%v user_sessions=%v user_resource_scopes=%v", userCheck != nil, sessionCheck != nil, scopeCheck != nil)
	}

	assertContainsAll(t, userCheck.required, []string{"username", "email", "account_status", "mobile_verified", "email_verified", "last_login_at"})
	assertContainsAll(t, sessionCheck.required, []string{"refresh_token_hash", "device_id", "expires_at", "revoked_at"})
	assertContainsAll(t, scopeCheck.required, []string{"user_id", "domain", "store_id", "team_id"})
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
