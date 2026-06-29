package consul

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"goshop/gmicro/registry"
)

func TestRegisterUsesHTTPHealthCheckForHTTPEndpoints(t *testing.T) {
	var got struct {
		ID     string `json:"ID"`
		Name   string `json:"Name"`
		Checks []struct {
			HTTP string `json:"HTTP"`
			TCP  string `json:"TCP"`
		} `json:"Checks"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/v1/agent/service/register" {
			t.Fatalf("request = %s %s, want PUT /v1/agent/service/register", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode register request failed: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	apiClient, err := api.NewClient(&api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	client := NewClient(apiClient)
	client.heartbeat = false

	err = client.Register(context.Background(), &registry.ServiceInstance{
		ID:        "goods-1",
		Name:      "goods",
		Version:   "v1",
		Endpoints: []string{"grpc://127.0.0.1:9000", "http://127.0.0.1:8000"},
	}, true)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	var hasHTTP bool
	var hasHTTPAsTCP bool
	for _, check := range got.Checks {
		if check.HTTP == "http://127.0.0.1:8000/readyz" {
			hasHTTP = true
		}
		if check.TCP == "127.0.0.1:8000" {
			hasHTTPAsTCP = true
		}
	}
	if !hasHTTP {
		t.Fatalf("registered checks = %+v, want HTTP /readyz check", got.Checks)
	}
	if hasHTTPAsTCP {
		t.Fatalf("registered checks = %+v, HTTP endpoint should not use TCP check", got.Checks)
	}
}

func TestRegisterUsesContext(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})

	apiClient, err := api.NewClient(&api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	client := NewClient(apiClient)
	client.heartbeat = false

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = client.Register(ctx, &registry.ServiceInstance{
		ID:        "goods-1",
		Name:      "goods",
		Endpoints: []string{"grpc://127.0.0.1:9000"},
	}, true)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Register() error = %v, want context deadline exceeded", err)
	}
}

func TestDeregisterUsesContext(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})

	apiClient, err := api.NewClient(&api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	client := NewClient(apiClient)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = client.Deregister(ctx, "goods-1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Deregister() error = %v, want context deadline exceeded", err)
	}
}

func TestRegisterRejectsEndpointWithoutPort(t *testing.T) {
	apiClient, err := api.NewClient(&api.Config{Address: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	client := NewClient(apiClient)
	client.heartbeat = false

	err = client.Register(context.Background(), &registry.ServiceInstance{
		ID:        "goods-1",
		Name:      "goods",
		Version:   "v1",
		Endpoints: []string{"http://127.0.0.1"},
	}, true)
	if err == nil {
		t.Fatal("Register() error = nil, want missing port error")
	}
	if !strings.Contains(err.Error(), "missing port") {
		t.Fatalf("Register() error = %v, want missing port error", err)
	}
}

func TestRegisterAllowsCustomHTTPHealthCheckPath(t *testing.T) {
	var got struct {
		Checks []struct {
			HTTP string `json:"HTTP"`
		} `json:"Checks"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode register request failed: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	apiClient, err := api.NewClient(&api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	client := NewClient(apiClient)
	client.heartbeat = false
	client.httpHealthCheckPath = "/healthz"

	err = client.Register(context.Background(), &registry.ServiceInstance{
		ID:        "goods-1",
		Name:      "goods",
		Version:   "v1",
		Endpoints: []string{"http://127.0.0.1:8000/api"},
	}, true)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	if len(got.Checks) != 1 || got.Checks[0].HTTP != "http://127.0.0.1:8000/healthz" {
		t.Fatalf("registered checks = %+v, want custom HTTP health check path", got.Checks)
	}
}
