package discovery

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"goshop/gmicro/registry"

	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

type recordingClientConn struct {
	resolver.ClientConn
	mu      sync.Mutex
	states  []resolver.State
	updates chan struct{}
}

func (c *recordingClientConn) UpdateState(state resolver.State) error {
	c.mu.Lock()
	c.states = append(c.states, state)
	c.mu.Unlock()
	if c.updates != nil {
		select {
		case c.updates <- struct{}{}:
		default:
		}
	}
	return nil
}

func (c *recordingClientConn) stateSnapshot() []resolver.State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]resolver.State(nil), c.states...)
}

func (c *recordingClientConn) ReportError(error) {}

func (c *recordingClientConn) NewAddress([]resolver.Address) {}

func (c *recordingClientConn) ParseServiceConfig(string) *serviceconfig.ParseResult {
	return nil
}

func TestUpdateRetainsPreviousStateDuringEmptySnapshotGracePeriod(t *testing.T) {
	cc := &recordingClientConn{}
	r := &discoveryResolver{cc: cc, insecure: true, emptySnapshotGracePeriod: time.Hour}
	t.Cleanup(r.Close)

	r.update([]*registry.ServiceInstance{
		{
			Name:      "goods",
			Endpoints: []string{"grpc://127.0.0.1:8080"},
		},
	})
	r.update(nil)

	states := cc.stateSnapshot()
	if len(states) != 1 {
		t.Fatalf("UpdateState calls = %d, want 1", len(states))
	}
	if len(states[0].Addresses) != 1 || states[0].Addresses[0].Addr != "127.0.0.1:8080" {
		t.Fatalf("UpdateState addresses = %v, want the previous valid address", states[0].Addresses)
	}
}

func TestUpdateDoesNotForceTLSOverrideFromServiceName(t *testing.T) {
	cc := &recordingClientConn{}
	r := &discoveryResolver{cc: cc}
	t.Cleanup(r.Close)

	r.update([]*registry.ServiceInstance{
		{
			Name:      "goods-srv",
			Endpoints: []string{"grpc://127.0.0.1:9000?isSecure=true"},
		},
	})

	states := cc.stateSnapshot()
	if len(states) != 1 || len(states[0].Addresses) != 1 {
		t.Fatalf("UpdateState addresses = %v, want one address", states)
	}
	if got := states[0].Addresses[0].ServerName; got != "" {
		t.Fatalf("resolver address server name = %q, want empty so client TLS config can apply", got)
	}
}

func TestUpdateUsesMetadataTLSServerNameOverride(t *testing.T) {
	cc := &recordingClientConn{}
	r := &discoveryResolver{cc: cc}
	t.Cleanup(r.Close)

	r.update([]*registry.ServiceInstance{
		{
			Name:      "goods-srv",
			Endpoints: []string{"grpc://127.0.0.1:9000?isSecure=true"},
			Metadata: map[string]string{
				"tls_server_name": "goshop.internal",
			},
		},
	})

	states := cc.stateSnapshot()
	if len(states) != 1 || len(states[0].Addresses) != 1 {
		t.Fatalf("UpdateState addresses = %v, want one address", states)
	}
	if got := states[0].Addresses[0].ServerName; got != "goshop.internal" {
		t.Fatalf("resolver address server name = %q, want goshop.internal", got)
	}
}

type failingClientConn struct {
	recordingClientConn
}

func (c *failingClientConn) UpdateState(state resolver.State) error {
	_ = c.recordingClientConn.UpdateState(state)
	return errors.New("update failed")
}

func TestUpdatePublishesEmptyStateAfterEmptySnapshotGracePeriod(t *testing.T) {
	cc := &recordingClientConn{updates: make(chan struct{}, 2)}
	r := &discoveryResolver{cc: cc, insecure: true, emptySnapshotGracePeriod: 10 * time.Millisecond}
	t.Cleanup(r.Close)

	r.update([]*registry.ServiceInstance{{
		Name:      "goods",
		Endpoints: []string{"grpc://127.0.0.1:8080"},
	}})
	r.update(nil)

	<-cc.updates // Initial non-empty resolver state.
	select {
	case <-cc.updates:
	case <-time.After(time.Second):
		states := cc.stateSnapshot()
		t.Fatalf("UpdateState calls = %d, want 2 after empty snapshot grace period", len(states))
	}
	states := cc.stateSnapshot()
	if got := len(states[1].Addresses); got != 0 {
		t.Fatalf("empty resolver state addresses = %d, want 0", got)
	}
}

func TestUpdateDefersEmptyStateWhenNoEndpoints(t *testing.T) {
	cc := &failingClientConn{}
	r := &discoveryResolver{cc: cc, insecure: true, emptySnapshotGracePeriod: time.Hour}
	t.Cleanup(r.Close)

	r.update(nil)

	states := cc.stateSnapshot()
	if len(states) != 0 {
		t.Fatalf("UpdateState calls = %d, want 0", len(states))
	}
}

func TestBuilderBuildTimesOutAndCancelsWatch(t *testing.T) {
	d := &blockingDiscovery{
		started: make(chan string, 1),
		done:    make(chan error, 1),
	}
	b := NewBuilder(d, WithTimeout(10*time.Millisecond))

	r, err := b.Build(
		resolver.Target{URL: url.URL{Path: "/goods"}},
		&recordingClientConn{},
		resolver.BuildOptions{},
	)
	if err == nil {
		r.Close()
		t.Fatal("Build() error = nil, want timeout")
	}
	if !strings.Contains(err.Error(), "overtime") {
		t.Fatalf("Build() error = %v, want overtime", err)
	}
	if got := <-d.started; got != "goods" {
		t.Fatalf("Watch() service name = %q, want goods", got)
	}
	select {
	case err := <-d.done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Watch() context error = %v, want canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Watch() was not canceled after build timeout")
	}
}

type blockingDiscovery struct {
	started chan string
	done    chan error
}

func (d *blockingDiscovery) GetService(context.Context, string) ([]*registry.ServiceInstance, error) {
	return nil, nil
}

func (d *blockingDiscovery) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	d.started <- serviceName
	<-ctx.Done()
	d.done <- ctx.Err()
	return nil, ctx.Err()
}
