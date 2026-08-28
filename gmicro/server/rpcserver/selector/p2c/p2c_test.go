package p2c

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"goshop/gmicro/server/rpcserver/selector"
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
func (n *testNode) Version() string             { return "v1" }
func (n *testNode) Metadata() map[string]string { return nil }
func (n *testNode) Raw() selector.Node          { return n }
func (n *testNode) Weight() float64             { return n.weight }
func (n *testNode) PickElapsed() time.Duration  { return n.elapsed }
func (n *testNode) Pick() selector.DoneFunc {
	n.picks++
	return func(context.Context, selector.DoneInfo) {}
}

func TestBalancerPick(t *testing.T) {
	tests := []struct {
		name    string
		nodes   []selector.WeightedNode
		want    string
		wantErr error
	}{
		{name: "no nodes", wantErr: selector.ErrNoAvailable},
		{name: "single node", nodes: []selector.WeightedNode{&testNode{address: "one", weight: 1}}, want: "one"},
		{name: "higher weight wins", nodes: []selector.WeightedNode{
			&testNode{address: "low", weight: 1},
			&testNode{address: "high", weight: 10},
		}, want: "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balancer := &Balancer{r: rand.New(rand.NewSource(1))}
			got, done, err := balancer.Pick(context.Background(), tt.nodes)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Pick() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Address() != tt.want {
				t.Fatalf("Pick() selected %q, want %q", got.Address(), tt.want)
			}
			if done == nil {
				t.Fatal("Pick() done = nil")
			}
			done(context.Background(), selector.DoneInfo{})
		})
	}
}

func TestBalancerPickForcesStaleNode(t *testing.T) {
	stale := &testNode{address: "stale", weight: 1, elapsed: forcePick + time.Second}
	fresh := &testNode{address: "fresh", weight: 100}
	balancer := &Balancer{r: rand.New(rand.NewSource(1))}

	got, _, err := balancer.Pick(context.Background(), []selector.WeightedNode{stale, fresh})
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	if got != stale {
		t.Fatalf("Pick() = %q, want stale node", got.Address())
	}
}
