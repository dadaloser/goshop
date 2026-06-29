package discovery

import (
	"errors"
	"testing"

	"goshop/gmicro/registry"

	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

type recordingClientConn struct {
	resolver.ClientConn
	states []resolver.State
}

func (c *recordingClientConn) UpdateState(state resolver.State) error {
	c.states = append(c.states, state)
	return nil
}

func (c *recordingClientConn) ReportError(error) {}

func (c *recordingClientConn) NewAddress([]resolver.Address) {}

func (c *recordingClientConn) ParseServiceConfig(string) *serviceconfig.ParseResult {
	return nil
}

func TestUpdateWritesEmptyAddressState(t *testing.T) {
	cc := &recordingClientConn{}
	r := &discoveryResolver{cc: cc, insecure: true}

	r.update([]*registry.ServiceInstance{
		{
			Name:      "goods",
			Endpoints: []string{"http://127.0.0.1:8080"},
		},
	})

	if len(cc.states) != 1 {
		t.Fatalf("UpdateState calls = %d, want 1", len(cc.states))
	}
	if len(cc.states[0].Addresses) != 0 {
		t.Fatalf("UpdateState addresses = %v, want empty", cc.states[0].Addresses)
	}
}

type failingClientConn struct {
	recordingClientConn
}

func (c *failingClientConn) UpdateState(state resolver.State) error {
	_ = c.recordingClientConn.UpdateState(state)
	return errors.New("update failed")
}

func TestUpdateStillCallsClientConnWhenNoEndpoints(t *testing.T) {
	cc := &failingClientConn{}
	r := &discoveryResolver{cc: cc, insecure: true}

	r.update(nil)

	if len(cc.states) != 1 {
		t.Fatalf("UpdateState calls = %d, want 1", len(cc.states))
	}
}
