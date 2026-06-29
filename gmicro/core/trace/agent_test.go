package trace

import (
	"context"
	"testing"
)

func TestInitAgentReturnsExporterError(t *testing.T) {
	agents = make(map[string]struct{})

	err := InitAgent(Options{
		Name:     "test-service",
		Endpoint: string([]byte{0x7f}),
		Sampler:  1,
	})
	if err == nil {
		t.Fatal("InitAgent() error = nil, want exporter configuration error")
	}
}

func TestInitAgentAcceptsOTLPHostPortEndpoint(t *testing.T) {
	agents = make(map[string]struct{})

	err := InitAgent(Options{
		Name:     "test-service",
		Endpoint: "127.0.0.1:4318",
		Sampler:  1,
	})
	if err != nil {
		t.Fatalf("InitAgent() error = %v, want nil", err)
	}
}

func TestShutdownFlushesAgents(t *testing.T) {
	agents = make(map[string]struct{})
	providers = nil

	err := InitAgent(Options{
		Name:    "test-service",
		Sampler: 1,
	})
	if err != nil {
		t.Fatalf("InitAgent() error = %v, want nil", err)
	}

	if len(providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(providers))
	}
	if err := Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}
	if len(providers) != 0 {
		t.Fatalf("providers = %d, want 0 after shutdown", len(providers))
	}
}
