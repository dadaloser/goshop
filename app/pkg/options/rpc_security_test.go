package options

import "testing"

func TestRPCSecurityOptionsToPolicyCopiesMutableFields(t *testing.T) {
	opts := &RPCSecurityOptions{
		ServerName:              "service.example.test",
		AllowedClientIdentities: []string{"spiffe://example.test/client"},
	}

	policy := opts.ToPolicy()
	opts.AllowedClientIdentities[0] = "changed"

	if policy.ServerName != "service.example.test" {
		t.Fatalf("policy server name = %q", policy.ServerName)
	}
	if got := policy.AllowedClientIdentities[0]; got != "spiffe://example.test/client" {
		t.Fatalf("policy client identity = %q, want copied value", got)
	}
}
