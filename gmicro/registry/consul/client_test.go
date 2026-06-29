package consul

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

func TestRegisterHeartbeatUpdateUsesTimeout(t *testing.T) {
	updateDeadline := make(chan time.Time, 1)
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && req.URL.Path == "/v1/agent/service/register":
			return consulOKResponse(), nil
		case req.Method == http.MethodPut && strings.HasPrefix(req.URL.Path, "/v1/agent/check/update/"):
			deadline, ok := req.Context().Deadline()
			if !ok {
				t.Fatal("heartbeat update request has no context deadline")
			}
			updateDeadline <- deadline
			return consulOKResponse(), nil
		default:
			t.Fatalf("unexpected request = %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})

	apiClient, err := api.NewClient(&api.Config{
		Address:    "http://consul.local",
		HttpClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	client := NewClient(apiClient)
	client.healthcheckInterval = 0
	client.heartbeatTimeout = 20 * time.Millisecond
	t.Cleanup(client.cancel)

	err = client.Register(context.Background(), &registry.ServiceInstance{
		ID:        "goods-1",
		Name:      "goods",
		Version:   "v1",
		Endpoints: []string{"grpc://127.0.0.1:9000"},
	}, false)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	select {
	case deadline := <-updateDeadline:
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > client.heartbeatTimeout {
			t.Fatalf("heartbeat deadline remaining = %v, want within %v", remaining, client.heartbeatTimeout)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("heartbeat update request was not sent")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func consulOKResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(nil)),
		Header:     make(http.Header),
	}
}
