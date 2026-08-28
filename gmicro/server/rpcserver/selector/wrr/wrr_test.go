package wrr

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
}

func (n *testNode) Scheme() string              { return "grpc" }
func (n *testNode) Address() string             { return n.address }
func (n *testNode) ServiceName() string         { return "test" }
func (n *testNode) InitialWeight() *int64       { return nil }
func (n *testNode) Version() string             { return "" }
func (n *testNode) Metadata() map[string]string { return nil }
func (n *testNode) Raw() selector.Node          { return n }
func (n *testNode) Weight() float64             { return n.weight }
func (n *testNode) PickElapsed() time.Duration  { return 0 }
func (n *testNode) Pick() selector.DoneFunc     { return func(context.Context, selector.DoneInfo) {} }

func TestBalancerPick(t *testing.T) {
	balancer := (&Builder{}).Build()
	if _, _, err := balancer.Pick(t.Context(), nil); !errors.Is(err, selector.ErrNoAvailable) {
		t.Fatalf("Pick() error = %v, want %v", err, selector.ErrNoAvailable)
	}

	light := &testNode{address: "light", weight: 1}
	heavy := &testNode{address: "heavy", weight: 3}
	picks := map[string]int{}
	for range 4 {
		node, done, err := balancer.Pick(t.Context(), []selector.WeightedNode{light, heavy})
		if err != nil {
			t.Fatalf("Pick() error = %v", err)
		}
		picks[node.Address()]++
		done(t.Context(), selector.DoneInfo{})
	}
	if picks["light"] != 1 || picks["heavy"] != 3 {
		t.Fatalf("weighted picks = %#v, want light=1 heavy=3", picks)
	}
}
