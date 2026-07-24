package payment

import (
	"context"
	"testing"
	"time"

	"goshop/pkg/storage"
)

func TestRedisNonceStoreRejectsInvalidInput(t *testing.T) {
	store := NewRedisNonceStore()
	tests := []struct {
		name  string
		nonce string
		ttl   time.Duration
	}{
		{name: "empty nonce", ttl: time.Minute},
		{name: "zero ttl", nonce: "nonce-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reserved, err := store.Reserve(context.Background(), tt.nonce, tt.ttl)
			if err == nil || reserved {
				t.Fatalf("Reserve() reserved=%v err=%v, want validation error", reserved, err)
			}
		})
	}
}

func TestRedisNonceStoreReportsRedisDown(t *testing.T) {
	store := NewRedisNonceStore()
	reserved, err := store.Reserve(context.Background(), "nonce-1", time.Minute)
	if err != storage.ErrRedisIsDown || reserved {
		t.Fatalf("Reserve() reserved=%v err=%v, want ErrRedisIsDown", reserved, err)
	}
}
