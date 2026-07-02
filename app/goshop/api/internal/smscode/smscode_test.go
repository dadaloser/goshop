package smscode

import "testing"

func TestRegisterKeyIncludesTypeAndMobile(t *testing.T) {
	got := RegisterKey("13800138000")
	want := "sms:1:13800138000"
	if got != want {
		t.Fatalf("RegisterKey() = %q, want %q", got, want)
	}
}
