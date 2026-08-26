package storage

import (
	"context"
	"strings"
	"testing"
)

func TestServerStartRejectsInvalidConfig(t *testing.T) {
	server := NewServer(&Config{})
	err := server.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "host or address") {
		t.Fatalf("Start() error = %v, want missing Redis address error", err)
	}
}
