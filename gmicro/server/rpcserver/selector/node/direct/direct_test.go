package direct

import (
	"context"
	"testing"
	"time"

	"goshop/gmicro/registry"
	"goshop/gmicro/server/rpcserver/selector"
)

func TestNodeWeightAndPick(t *testing.T) {
	defaultNode := (&Builder{}).Build(selector.NewNode("grpc", "127.0.0.1:9000", nil))
	if got := defaultNode.Weight(); got != defaultWeight {
		t.Fatalf("default Weight() = %v, want %v", got, defaultWeight)
	}

	weightedNode := (&Builder{}).Build(selector.NewNode("grpc", "127.0.0.1:9001", &registry.ServiceInstance{
		Metadata: map[string]string{"weight": "42"},
	}))
	if got := weightedNode.Weight(); got != 42 {
		t.Fatalf("configured Weight() = %v, want 42", got)
	}
	if weightedNode.PickElapsed() <= 0 {
		t.Fatal("PickElapsed() before Pick() must be positive")
	}
	done := weightedNode.Pick()
	if done == nil {
		t.Fatal("Pick() done = nil")
	}
	done(context.Background(), selector.DoneInfo{})
	if got := weightedNode.PickElapsed(); got < 0 || got > time.Second {
		t.Fatalf("PickElapsed() after Pick() = %v, want [0, 1s]", got)
	}
}
