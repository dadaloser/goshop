package ewma

import (
	"context"
	"errors"
	"testing"
	"time"

	"goshop/gmicro/server/rpcserver/selector"
)

func TestNodePickTracksInflightAndHealth(t *testing.T) {
	node := (&Builder{}).Build(selector.NewNode("grpc", "127.0.0.1:9000", nil)).(*Node)
	if node.Weight() <= 0 {
		t.Fatalf("initial Weight() = %v, want positive", node.Weight())
	}

	done := node.Pick()
	if got := node.inflight; got != 2 {
		t.Fatalf("inflight after Pick() = %d, want 2", got)
	}
	done(context.Background(), selector.DoneInfo{Err: context.DeadlineExceeded})
	if got := node.inflight; got != 1 {
		t.Fatalf("inflight after Done() = %d, want 1", got)
	}
	if got := node.health(); got >= 1000 {
		t.Fatalf("health after deadline error = %d, want degraded health", got)
	}
	if elapsed := node.PickElapsed(); elapsed < 0 || elapsed > time.Second {
		t.Fatalf("PickElapsed() = %v, want [0, 1s]", elapsed)
	}
}

func TestNodeCustomErrorHandlerControlsHealth(t *testing.T) {
	node := (&Builder{ErrHandler: func(err error) bool {
		return errors.Is(err, errRetryable)
	}}).Build(selector.NewNode("grpc", "127.0.0.1:9000", nil)).(*Node)

	node.Pick()(context.Background(), selector.DoneInfo{Err: errRetryable})
	if got := node.health(); got >= 1000 {
		t.Fatalf("health after retryable error = %d, want degraded health", got)
	}
}

var errRetryable = errors.New("retryable")
