package app

import (
	"encoding/json"
	"testing"
)

func TestRedactJSON(t *testing.T) {
	raw := `{"mysql":{"password":"pw","host":"127.0.0.1"},"jwt":{"key":"secret-key"},"server":{"profiling-token":"token","port":8080}}`

	got := RedactJSON(raw)

	var decoded map[string]map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("RedactJSON() returned invalid JSON: %v", err)
	}
	if decoded["mysql"]["password"] != redactedValue {
		t.Fatalf("mysql.password = %v, want redacted", decoded["mysql"]["password"])
	}
	if decoded["jwt"]["key"] != redactedValue {
		t.Fatalf("jwt.key = %v, want redacted", decoded["jwt"]["key"])
	}
	if decoded["server"]["profiling-token"] != redactedValue {
		t.Fatalf("server.profiling-token = %v, want redacted", decoded["server"]["profiling-token"])
	}
	if decoded["mysql"]["host"] != "127.0.0.1" {
		t.Fatalf("mysql.host = %v, want unchanged host", decoded["mysql"]["host"])
	}
}

func TestRedactJSONInvalidInput(t *testing.T) {
	const raw = "not-json"

	if got := RedactJSON(raw); got != raw {
		t.Fatalf("RedactJSON() = %q, want original input", got)
	}
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{name: "password", key: "mysql.password", want: true},
		{name: "token", key: "server.profiling-token", want: true},
		{name: "api secret", key: "sms.api_secret", want: true},
		{name: "normal host", key: "mysql.host", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSensitiveKey(tt.key); got != tt.want {
				t.Fatalf("isSensitiveKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}
