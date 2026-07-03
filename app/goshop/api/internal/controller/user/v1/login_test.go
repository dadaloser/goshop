package user

import "testing"

func TestIsLoginUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{name: "mobile", username: "13800138000", want: true},
		{name: "email", username: "user@example.com", want: true},
		{name: "empty", username: "", want: false},
		{name: "bad mobile", username: "12345", want: false},
		{name: "bad email", username: "user@", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLoginUsername(tt.username); got != tt.want {
				t.Fatalf("isLoginUsername(%q) = %t, want %t", tt.username, got, tt.want)
			}
		})
	}
}
