package smslimit

import (
	"strings"
	"testing"
)

func TestKeyedMobileHashesMobile(t *testing.T) {
	got, err := cooldownKey(" 13800138000 ", 1)
	if err != nil {
		t.Fatalf("cooldownKey() error = %v", err)
	}
	if !strings.HasPrefix(got, cooldownKeyPrefix) {
		t.Fatalf("cooldownKey() = %q, want prefix %q", got, cooldownKeyPrefix)
	}
	if strings.Contains(got, "13800138000") {
		t.Fatal("cooldownKey() leaked raw mobile")
	}
}

func TestKeyedMobileIncludesCodeType(t *testing.T) {
	registerKey, err := cooldownKey("13800138000", 1)
	if err != nil {
		t.Fatalf("cooldownKey(register) error = %v", err)
	}
	loginKey, err := cooldownKey("13800138000", 2)
	if err != nil {
		t.Fatalf("cooldownKey(login) error = %v", err)
	}
	if registerKey == loginKey {
		t.Fatal("cooldownKey() returned same key for different code types")
	}
}

func TestKeyedMobileRejectsEmptyMobile(t *testing.T) {
	if _, err := windowKey(" ", 1); err != ErrEmptyMobile {
		t.Fatalf("windowKey() error = %v, want ErrEmptyMobile", err)
	}
}
