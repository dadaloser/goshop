package app

import (
	"context"
	"errors"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"goshop/gmicro/core/trace"
	"goshop/gmicro/registry"
)

func TestWithRegistrarTimeout(t *testing.T) {
	app := New(WithRegistrarTimeout(3 * time.Second))

	if app.opts.registrarTimeout != 3*time.Second {
		t.Fatalf("registrarTimeout = %v, want 3s", app.opts.registrarTimeout)
	}
}

func TestWithStopTimeout(t *testing.T) {
	app := New(WithStopTimeout(15 * time.Second))

	if app.opts.stopTimeout != 15*time.Second {
		t.Fatalf("stopTimeout = %v, want 15s", app.opts.stopTimeout)
	}
}

type fakeServer struct {
	ready   chan struct{}
	started chan struct{}
	stopped int32
}

func newFakeServer() *fakeServer {
	return &fakeServer{
		ready:   make(chan struct{}),
		started: make(chan struct{}),
	}
}

func (s *fakeServer) Ready() <-chan struct{} {
	return s.ready
}

func (s *fakeServer) Start(ctx context.Context) error {
	close(s.started)
	close(s.ready)
	<-ctx.Done()
	return nil
}

func (s *fakeServer) Stop(context.Context) error {
	atomic.StoreInt32(&s.stopped, 1)
	return nil
}

type endpointFakeServer struct {
	*fakeServer
	endpoint *url.URL
}

func (s *endpointFakeServer) Endpoint() *url.URL {
	return s.endpoint
}

type failingRegistrar struct{}

func (failingRegistrar) Register(context.Context, *registry.ServiceInstance) error {
	return errors.New("register failed")
}

func (failingRegistrar) Deregister(context.Context, *registry.ServiceInstance) error {
	return nil
}

func TestRunStopsStartedServersWhenRegisterFails(t *testing.T) {
	base := newFakeServer()
	srv := &endpointFakeServer{
		fakeServer: base,
		endpoint:   &url.URL{Scheme: "grpc", Host: "127.0.0.1:9000"},
	}
	app := New(
		WithName("server-1"),
		WithRegistrar(failingRegistrar{}),
		WithRegistrarTimeout(time.Second),
		WithStopTimeout(time.Second),
		WithServer(srv),
		WithEndpoints([]*url.URL{srv.Endpoint()}),
	)

	err := app.RunContext(context.Background())
	if err == nil {
		t.Fatal("Run() error = nil, want register error")
	}
	if atomic.LoadInt32(&srv.stopped) != 1 {
		t.Fatal("server was not stopped after register failure")
	}
}

func TestStopFlushesTraceProviders(t *testing.T) {
	traceAgents := make(map[string]struct{})
	_ = traceAgents
	if err := trace.InitAgent(trace.Options{Name: "test-service", Sampler: 1}); err != nil {
		t.Fatalf("InitAgent() error = %v, want nil", err)
	}

	app := New(WithStopTimeout(time.Second))
	if err := app.Stop(); err != nil {
		t.Fatalf("Stop() error = %v, want nil", err)
	}
	if trace.ProviderCount() != 0 {
		t.Fatalf("trace providers = %d, want 0 after app stop", trace.ProviderCount())
	}
}
