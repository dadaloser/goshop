package tokenrevocation

import (
	"strings"
	"testing"
)

func TestKeyHashesToken(t *testing.T) {
	got, err := key(" raw-token ")
	if err != nil {
		t.Fatalf("key() error = %v", err)
	}
	if !strings.HasPrefix(got, keyPrefix) {
		t.Fatalf("key() = %q, want prefix %q", got, keyPrefix)
	}
	if strings.Contains(got, "raw-token") {
		t.Fatal("key() leaked raw token")
	}
}

func TestKeyRejectsEmptyToken(t *testing.T) {
	if _, err := key(" "); err != ErrEmptyToken {
		t.Fatalf("key() error = %v, want ErrEmptyToken", err)
	}
}
