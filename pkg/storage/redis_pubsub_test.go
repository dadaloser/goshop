package storage

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestHandlePubSubMessagesReturnsWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	messages := make(chan *redis.Message)
	result := make(chan error, 1)
	go func() {
		result <- handlePubSubMessages(ctx, messages, func(interface{}) {})
	}()
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("handlePubSubMessages() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("handlePubSubMessages() did not return after context cancellation")
	}
}
