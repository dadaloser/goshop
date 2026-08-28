package wrr

import (
	"context"
	"errors"
	"testing"
	"time"

	"goshop/gmicro/server/rpcserver/selector"
)

type testNode struct {
	address string
	weight  float64
}

func (n *testNode) Scheme() string              { return "grpc" }
func (n *testNode) Address() string             { return n.address }
func (n *testNode) ServiceName() string         { return "test" }
func (n *testNode) InitialWeight() *int64       { return nil }
func (n *testNode) Version() string             { return "v1" }
func (n *testNode) Metadata() map[string]string { return nil }
func (n *testNode) Raw() selector.Node          { return n }
func (n *testNode) Weight() float64             { return n.weight }
func (n *testNode) PickElapsed() time.Duration  { return 0 }
func (n *testNode) Pick() selector.DoneFunc     { return func(context.Context, selector.DoneInfo) {} }

func TestBalancerPick(t *testing.T) {
	balancer := &Balancer{currentWeight: make(map[string]float64)}
	_, _, err := balancer.Pick(context.Background(), nil)
	if !errors.Is(err, selector.ErrNoAvailable) {
		t.Fatalf("Pick() error = %v, want %v", err, selector.ErrNoAvailable)
	}

	nodes := []selector.WeightedNode{
		&testNode{address: "heavy", weight: 5},
		&testNode{address: "light", weight: 1},
	}
	counts := map[string]int{}
	for range 6 {
		got, done, err := balancer.Pick(context.Background(), nodes)
		if err != nil {
			t.Fatalf("Pick() error = %v", err)
		}
		counts[got.Address()]++
		done(context.Background(), selector.DoneInfo{})
	}
	if counts["heavy"] != 5 || counts["light"] != 1 {
		t.Fatalf("weighted picks = %#v, want heavy=5 light=1", counts)
	}
}
