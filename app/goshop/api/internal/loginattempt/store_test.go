package loginattempt

import (
	"strings"
	"testing"
)

func TestKeyedIdentifierHashesIdentifier(t *testing.T) {
	got, err := failKey(" User@Example.COM ")
	if err != nil {
		t.Fatalf("failKey() error = %v", err)
	}
	if !strings.HasPrefix(got, failKeyPrefix) {
		t.Fatalf("failKey() = %q, want prefix %q", got, failKeyPrefix)
	}
	if strings.Contains(strings.ToLower(got), "user@example.com") {
		t.Fatal("failKey() leaked raw identifier")
	}
}

func TestKeyedIdentifierRejectsEmptyIdentifier(t *testing.T) {
	if _, err := lockKey(" "); err != ErrEmptyIdentifier {
		t.Fatalf("lockKey() error = %v, want ErrEmptyIdentifier", err)
	}
}
