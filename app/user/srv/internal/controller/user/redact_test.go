package user

import "testing"

func TestRedactMobileForLog(t *testing.T) {
	tests := []struct {
		name   string
		mobile string
		want   string
	}{
		{name: "mobile", mobile: "13800138000", want: "138****8000"},
		{name: "trimmed", mobile: " 13800138000 ", want: "138****8000"},
		{name: "short", mobile: "12345", want: "***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactMobileForLog(tt.mobile); got != tt.want {
				t.Fatalf("redactMobileForLog() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRedactEmailForLog(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{name: "email", email: "user@example.com", want: "u***@example.com"},
		{name: "single local char", email: "u@example.com", want: "*@example.com"},
		{name: "invalid", email: "not-email", want: "***"},
		{name: "empty", email: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactEmailForLog(tt.email); got != tt.want {
				t.Fatalf("redactEmailForLog() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRedactIdentifierForLog(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{name: "mobile", identifier: "13800138000", want: "138****8000"},
		{name: "email", identifier: "user@example.com", want: "u***@example.com"},
		{name: "username", identifier: "user_001", want: "u***1"},
		{name: "short", identifier: "ab", want: "***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactIdentifierForLog(tt.identifier); got != tt.want {
				t.Fatalf("redactIdentifierForLog() = %q, want %q", got, tt.want)
			}
		})
	}
}
