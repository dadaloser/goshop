package v1

import (
	"context"
	"testing"
	"unicode"
)

func TestSendSmsReturnsErrorWhenOptionsMissing(t *testing.T) {
	srv := NewSmsService(nil)

	if err := srv.SendSms(context.Background(), "13800138000", "template", "{}"); err == nil {
		t.Fatal("SendSms() error = nil, want error")
	}
}

func TestGenerateSmsCodeReturnsDigitsWithRequestedWidth(t *testing.T) {
	code, err := GenerateSmsCode(6)
	if err != nil {
		t.Fatalf("GenerateSmsCode() error = %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("GenerateSmsCode() len = %d, want 6", len(code))
	}
	for _, r := range code {
		if !unicode.IsDigit(r) {
			t.Fatalf("GenerateSmsCode() contains non-digit rune %q in %q", r, code)
		}
	}
}
