package p2c

import (
	"context"
	"errors"
	"testing"
	"time"

	selector "goshop/gmicro/server/rpcserver/selector"
)

type testNode struct {
	address string
	weight  float64
	elapsed time.Duration
	picks   int
}

func (n *testNode) Scheme() string              { return "grpc" }
func (n *testNode) Address() string             { return n.address }
func (n *testNode) ServiceName() string         { return "test" }
func (n *testNode) InitialWeight() *int64       { return nil }
func (n *testNode) Version() string             { return "" }
func (n *testNode) Metadata() map[string]string { return nil }
func (n *testNode) Raw() selector.Node          { return n }
func (n *testNode) Weight() float64             { return n.weight }
func (n *testNode) PickElapsed() time.Duration  { return n.elapsed }
func (n *testNode) Pick() selector.DoneFunc {
	n.picks++
	return func(context.Context, selector.DoneInfo) {}
}

func TestBalancerPick(t *testing.T) {
	t.Run("no nodes", func(t *testing.T) {
		balancer := (&Builder{}).Build()
		_, _, err := balancer.Pick(t.Context(), nil)
		if !errors.Is(err, selector.ErrNoAvailable) {
			t.Fatalf("Pick() error = %v, want %v", err, selector.ErrNoAvailable)
		}
	})

	t.Run("prefers higher weight", func(t *testing.T) {
		low := &testNode{address: "low", weight: 1}
		high := &testNode{address: "high", weight: 10}
		balancer := (&Builder{}).Build()

		got, done, err := balancer.Pick(t.Context(), []selector.WeightedNode{low, high})
		if err != nil {
			t.Fatalf("Pick() error = %v", err)
		}
		if got != high {
			t.Fatalf("Pick() = %s, want high", got.Address())
		}
		if done == nil {
			t.Fatal("Pick() done = nil")
		}
		done(t.Context(), selector.DoneInfo{})
	})

	t.Run("forces stale lower weight node", func(t *testing.T) {
		stale := &testNode{address: "stale", weight: 1, elapsed: forcePick + time.Nanosecond}
		fresh := &testNode{address: "fresh", weight: 10}
		balancer := (&Builder{}).Build()

		got, _, err := balancer.Pick(t.Context(), []selector.WeightedNode{stale, fresh})
		if err != nil {
			t.Fatalf("Pick() error = %v", err)
		}
		if got != stale {
			t.Fatalf("Pick() = %s, want stale node", got.Address())
		}
	})
}
