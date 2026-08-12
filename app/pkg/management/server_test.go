package management

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"goshop/app/pkg/options"
)

func TestNewServerExposesHealthAndMetrics(t *testing.T) {
	port := availablePort(t)
	opts := options.NewServerOptions()
	opts.Name = "management-test"
	opts.Host = "127.0.0.1"
	opts.HttpPort = port

	server, err := NewServer(opts)
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	startErr := make(chan error, 1)
	go func() {
		startErr <- server.Start(context.Background())
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Stop(ctx); err != nil {
			t.Errorf("Server.Stop() error = %v, want nil", err)
		}
		select {
		case err := <-startErr:
			if err != nil {
				t.Errorf("Server.Start() error = %v, want nil", err)
			}
		case <-time.After(time.Second):
			t.Error("Server.Start() did not return after Stop()")
		}
	})

	select {
	case <-server.Ready():
	case <-time.After(time.Second):
		t.Fatal("Server.Ready() did not close")
	}

	client := &http.Client{Timeout: time.Second}
	for _, path := range []string{"/livez", "/readyz", "/metrics"} {
		resp, err := client.Get("http://" + server.Endpoint().Host + path)
		if err != nil {
			t.Errorf("GET %s error = %v, want nil", path, err)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, resp.StatusCode, http.StatusOK)
		}
	}
}

func TestNewServerRejectsMissingHTTPPort(t *testing.T) {
	opts := options.NewServerOptions()
	opts.HttpPort = 0

	if _, err := NewServer(opts); err == nil {
		t.Fatal("NewServer() error = nil, want missing HTTP port error")
	}
}

func availablePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v, want nil", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("listener.Close() error = %v, want nil", err)
	}
	return port
}
