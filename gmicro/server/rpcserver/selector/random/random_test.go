package random

import (
	"context"
	"errors"
	"testing"
	"time"

	selector "goshop/gmicro/server/rpcserver/selector"
)

type testNode struct{ address string }

func (n *testNode) Scheme() string              { return "grpc" }
func (n *testNode) Address() string             { return n.address }
func (n *testNode) ServiceName() string         { return "test" }
func (n *testNode) InitialWeight() *int64       { return nil }
func (n *testNode) Version() string             { return "" }
func (n *testNode) Metadata() map[string]string { return nil }
func (n *testNode) Raw() selector.Node          { return n }
func (n *testNode) Weight() float64             { return 1 }
func (n *testNode) PickElapsed() time.Duration  { return 0 }
func (n *testNode) Pick() selector.DoneFunc     { return func(context.Context, selector.DoneInfo) {} }

func TestBalancerPick(t *testing.T) {
	balancer := (&Builder{}).Build()
	if _, _, err := balancer.Pick(t.Context(), nil); !errors.Is(err, selector.ErrNoAvailable) {
		t.Fatalf("Pick() error = %v, want %v", err, selector.ErrNoAvailable)
	}

	first := &testNode{address: "first"}
	second := &testNode{address: "second"}
	for range 100 {
		node, done, err := balancer.Pick(t.Context(), []selector.WeightedNode{first, second})
		if err != nil {
			t.Fatalf("Pick() error = %v", err)
		}
		if node != first && node != second {
			t.Fatalf("Pick() returned unknown node %v", node)
		}
		if done == nil {
			t.Fatal("Pick() done = nil")
		}
		done(t.Context(), selector.DoneInfo{})
	}
}
