package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	appclient "goshop/app/pkg/client"
	appoptions "goshop/app/pkg/options"
	"goshop/gmicro/registry/consul"
	"goshop/gmicro/server/rpcserver"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestRunContextRealConsulDiscoveryMTLSClosedLoop(t *testing.T) {
	consulAddr := getenvDefault("CONSUL_SMOKE_ADDR", "192.168.1.92:8500")
	consulScheme := getenvDefault("CONSUL_SMOKE_SCHEME", "http")

	policy := newSmokeSecurityPolicy(t, "goshop.internal")
	rpcServer, err := rpcserver.NewServerE(
		rpcserver.WithAddress("0.0.0.0:0"),
		rpcserver.WithServerSecurityPolicy(policy),
	)
	if err != nil {
		t.Fatalf("NewServerE() error = %v", err)
	}

	cli, err := api.NewClient(&api.Config{
		Address: consulAddr,
		Scheme:  consulScheme,
	})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}

	register := consul.New(
		cli,
		consul.WithHealthCheck(true),
		consul.WithHeartbeat(true),
		consul.WithHealthCheckInterval(1),
	)

	serviceName := fmt.Sprintf("smoke-user-srv-%d", time.Now().UnixNano())
	smokeApp := New(
		WithName(serviceName),
		WithRPCServer(rpcServer),
		WithRegistrar(register),
		WithRegistrarTimeout(5*time.Second),
		WithStopTimeout(5*time.Second),
	)

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- smokeApp.RunContext(context.Background())
	}()

	entry, err := waitConsulPassingService(t.Context(), cli, serviceName, 15*time.Second)
	if err != nil {
		t.Fatalf("wait passing service %s: %v", serviceName, err)
	}
	if entry.Service == nil {
		t.Fatal("consul service entry = nil, want registered service")
	}

	grpcAddr, ok := entry.Service.TaggedAddresses["grpc"]
	if !ok {
		t.Fatalf("tagged addresses = %+v, want grpc endpoint", entry.Service.TaggedAddresses)
	}
	if grpcAddr.Address == "" || grpcAddr.Port == 0 {
		t.Fatalf("grpc tagged address = %+v, want endpoint address and port", grpcAddr)
	}
	if entry.Checks.AggregatedStatus() != api.HealthPassing {
		t.Fatalf("consul health status = %q, want passing", entry.Checks.AggregatedStatus())
	}

	foundServiceTTLCheck := false
	for _, check := range entry.Checks {
		if check.Type == "grpc" {
			t.Fatalf("health checks = %+v, secure grpc service should not register active grpc check under mTLS heartbeat mode", entry.Checks)
		}
		if strings.HasPrefix(check.CheckID, "service:") && check.Status == api.HealthPassing {
			foundServiceTTLCheck = true
		}
	}
	if !foundServiceTTLCheck {
		t.Fatalf("health checks = %+v, want passing service TTL heartbeat check", entry.Checks)
	}

	conn, err := appclient.DialService(
		context.Background(),
		&appoptions.RegistryOptions{
			Address: consulAddr,
			Scheme:  consulScheme,
		},
		policy,
		serviceName,
		rpcserver.WithConnectTimeout(3*time.Second),
		rpcserver.WithClientTimeout(3*time.Second),
	)
	if err != nil {
		t.Fatalf("DialService() error = %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	resp, err := healthClient.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health Check() error = %v", err)
	}
	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("health status = %s, want SERVING", resp.GetStatus())
	}

	if err := smokeApp.StopContext(context.Background()); err != nil {
		t.Fatalf("StopContext() error = %v", err)
	}

	if err := waitConsulDeregistered(t.Context(), cli, serviceName, 10*time.Second); err != nil {
		t.Fatalf("wait deregistered service %s: %v", serviceName, err)
	}

	select {
	case runErr := <-runErrCh:
		if runErr != nil && !errors.Is(runErr, context.Canceled) {
			t.Fatalf("RunContext() error = %v, want nil or context canceled", runErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunContext() did not return after StopContext")
	}
}

func waitConsulPassingService(ctx context.Context, cli *api.Client, serviceName string, timeout time.Duration) (*api.ServiceEntry, error) {
	deadline := time.Now().Add(timeout)
	var lastEntries []*api.ServiceEntry
	var lastErr error
	for {
		entries, _, err := cli.Health().Service(serviceName, "", true, new(api.QueryOptions).WithContext(ctx))
		if err == nil && len(entries) > 0 {
			return entries[0], nil
		}
		if err == nil {
			allEntries, _, allErr := cli.Health().Service(serviceName, "", false, new(api.QueryOptions).WithContext(ctx))
			if allErr == nil {
				lastEntries = allEntries
			}
		}
		lastErr = err
		if time.Now().After(deadline) {
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, fmt.Errorf("service did not become passing before timeout: %s", formatConsulEntries(lastEntries))
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func formatConsulEntries(entries []*api.ServiceEntry) string {
	if len(entries) == 0 {
		return "no service entries"
	}
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.Service == nil {
			parts = append(parts, "nil service entry")
			continue
		}
		checks := make([]string, 0, len(entry.Checks))
		for _, check := range entry.Checks {
			checks = append(checks, fmt.Sprintf("%s[%s]: %s", check.Type, check.Status, strings.TrimSpace(check.Output)))
		}
		parts = append(parts, fmt.Sprintf(
			"id=%s service=%s address=%s port=%d tagged=%v checks=%v",
			entry.Service.ID,
			entry.Service.Service,
			entry.Service.Address,
			entry.Service.Port,
			entry.Service.TaggedAddresses,
			checks,
		))
	}
	return strings.Join(parts, " | ")
}

func waitConsulDeregistered(ctx context.Context, cli *api.Client, serviceName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		entries, _, err := cli.Health().Service(serviceName, "", false, new(api.QueryOptions).WithContext(ctx))
		if err == nil && len(entries) == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return err
			}
			return fmt.Errorf("service still registered after timeout")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
