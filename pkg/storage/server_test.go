package storage

import (
	"strings"
	"testing"
)

func TestServerStartRejectsInvalidConfig(t *testing.T) {
	_, err := NewServer(&Config{})
	if err == nil || !strings.Contains(err.Error(), "host or address") {
		t.Fatalf("NewServer() error = %v, want missing Redis address error", err)
	}
}
