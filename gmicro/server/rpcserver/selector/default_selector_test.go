package selector

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testWeightedNodeBuilder struct {
	built int
}

func (b *testWeightedNodeBuilder) Build(n Node) WeightedNode {
	b.built++
	return &testWeightedNode{Node: n}
}

type testWeightedNode struct {
	Node
}

func (n *testWeightedNode) Raw() Node {
	return n.Node
}

func (n *testWeightedNode) Weight() float64 {
	return 1
}

func (n *testWeightedNode) Pick() DoneFunc {
	return func(context.Context, DoneInfo) {}
}

func (n *testWeightedNode) PickElapsed() time.Duration {
	return 0
}

func TestDefaultApplyReusesUnchangedWeightedNodes(t *testing.T) {
	builder := &testWeightedNodeBuilder{}
	sel := &Default{NodeBuilder: builder}

	first := NewNode("grpc", "127.0.0.1:9000", nil)
	sel.Apply([]Node{first})
	nodes, ok := sel.nodes.Load().([]WeightedNode)
	if !ok || len(nodes) != 1 {
		t.Fatalf("nodes = %v, want one weighted node", nodes)
	}
	weighted := nodes[0]

	second := NewNode("grpc", "127.0.0.1:9000", nil)
	sel.Apply([]Node{second})
	nodes, ok = sel.nodes.Load().([]WeightedNode)
	if !ok || len(nodes) != 1 {
		t.Fatalf("nodes = %v, want one weighted node after refresh", nodes)
	}
	if nodes[0] != weighted {
		t.Fatal("Apply rebuilt unchanged weighted node, want reuse")
	}
	if builder.built != 1 {
		t.Fatalf("Build() calls = %d, want 1", builder.built)
	}

	third := NewNode("grpc", "127.0.0.1:9001", nil)
	sel.Apply([]Node{third})
	nodes, ok = sel.nodes.Load().([]WeightedNode)
	if !ok || len(nodes) != 1 {
		t.Fatalf("nodes = %v, want one weighted node after address change", nodes)
	}
	if nodes[0] == weighted {
		t.Fatal("Apply reused changed weighted node, want rebuild")
	}
	if builder.built != 2 {
		t.Fatalf("Build() calls = %d, want 2", builder.built)
	}
}

type firstNodeBalancer struct{}

func (firstNodeBalancer) Pick(_ context.Context, nodes []WeightedNode) (WeightedNode, DoneFunc, error) {
	if len(nodes) == 0 {
		return nil, nil, ErrNoAvailable
	}
	return nodes[0], nodes[0].Pick(), nil
}

func TestDefaultSelect(t *testing.T) {
	sel := &Default{
		NodeBuilder: &testWeightedNodeBuilder{},
		Balancer:    firstNodeBalancer{},
	}
	if _, _, err := sel.Select(context.Background()); !errors.Is(err, ErrNoAvailable) {
		t.Fatalf("Select() error = %v, want %v", err, ErrNoAvailable)
	}

	node := NewNode("grpc", "127.0.0.1:9000", nil)
	sel.Apply([]Node{node})
	peer := &Peer{}
	ctx := NewPeerContext(context.Background(), peer)
	got, done, err := sel.Select(ctx)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if got != node {
		t.Fatalf("Select() node = %v, want %v", got, node)
	}
	if peer.Node != node {
		t.Fatalf("peer node = %v, want %v", peer.Node, node)
	}
	if done == nil {
		t.Fatal("Select() done = nil")
	}
	done(context.Background(), DoneInfo{})
}
