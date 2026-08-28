package random

import (
	"context"
	"errors"
	"testing"
	"time"

	"goshop/gmicro/server/rpcserver/selector"
)

type testNode struct{ address string }

func (n *testNode) Scheme() string              { return "grpc" }
func (n *testNode) Address() string             { return n.address }
func (n *testNode) ServiceName() string         { return "test" }
func (n *testNode) InitialWeight() *int64       { return nil }
func (n *testNode) Version() string             { return "v1" }
func (n *testNode) Metadata() map[string]string { return nil }
func (n *testNode) Raw() selector.Node          { return n }
func (n *testNode) Weight() float64             { return 1 }
func (n *testNode) PickElapsed() time.Duration  { return 0 }
func (n *testNode) Pick() selector.DoneFunc     { return func(context.Context, selector.DoneInfo) {} }

func TestBalancerPick(t *testing.T) {
	balancer := &Balancer{}
	_, _, err := balancer.Pick(context.Background(), nil)
	if !errors.Is(err, selector.ErrNoAvailable) {
		t.Fatalf("Pick() error = %v, want %v", err, selector.ErrNoAvailable)
	}

	nodes := []selector.WeightedNode{&testNode{address: "one"}, &testNode{address: "two"}}
	for range 20 {
		got, done, err := balancer.Pick(context.Background(), nodes)
		if err != nil {
			t.Fatalf("Pick() error = %v", err)
		}
		if got.Address() != "one" && got.Address() != "two" {
			t.Fatalf("Pick() selected unknown node %q", got.Address())
		}
		done(context.Background(), selector.DoneInfo{})
	}
}
