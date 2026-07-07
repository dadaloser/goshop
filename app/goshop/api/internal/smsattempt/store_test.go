package smsattempt

import (
	"strings"
	"testing"
)

func TestKeyedMobileHashesMobile(t *testing.T) {
	got, err := failKey(" 13800138000 ", 1)
	if err != nil {
		t.Fatalf("failKey() error = %v", err)
	}
	if !strings.HasPrefix(got, failKeyPrefix) {
		t.Fatalf("failKey() = %q, want prefix %q", got, failKeyPrefix)
	}
	if strings.Contains(got, "13800138000") {
		t.Fatal("failKey() leaked raw mobile")
	}
}

func TestKeyedMobileIncludesCodeType(t *testing.T) {
	registerKey, err := failKey("13800138000", 1)
	if err != nil {
		t.Fatalf("failKey(register) error = %v", err)
	}
	loginKey, err := failKey("13800138000", 2)
	if err != nil {
		t.Fatalf("failKey(login) error = %v", err)
	}
	if registerKey == loginKey {
		t.Fatal("failKey() returned same key for different code types")
	}
}

func TestKeyedMobileRejectsEmptyMobile(t *testing.T) {
	if _, err := lockKey(" ", 1); err != ErrEmptyMobile {
		t.Fatalf("lockKey() error = %v, want ErrEmptyMobile", err)
	}
}
