package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	appclient "goshop/app/pkg/client"
	appoptions "goshop/app/pkg/options"
	"goshop/gmicro/registry/consul"
	"goshop/gmicro/server/rpcserver"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestRunContextConsulDiscoveryMTLSClosedLoop(t *testing.T) {
	t.Parallel()

	policy := newSmokeSecurityPolicy(t, "goshop.internal")
	rpcServer, err := rpcserver.NewServerE(
		rpcserver.WithAddress("127.0.0.1:0"),
		rpcserver.WithServerSecurityPolicy(policy),
	)
	if err != nil {
		t.Fatalf("NewServerE() error = %v", err)
	}

	fakeConsul := newFakeConsulServer(t)
	consulClient, err := api.NewClient(&api.Config{Address: fakeConsul.server.URL})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	registry := consul.New(consulClient, consul.WithHealthCheck(true), consul.WithHeartbeat(false))

	smokeApp := New(
		WithName("smoke-user-srv"),
		WithRPCServer(rpcServer),
		WithRegistrar(registry),
		WithRegistrarTimeout(3*time.Second),
		WithStopTimeout(3*time.Second),
	)

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- smokeApp.RunContext(context.Background())
	}()

	registered := fakeConsul.waitRegister(t)
	grpcAddr, ok := registered.TaggedAddresses["grpc"]
	if !ok {
		t.Fatalf("register payload tagged addresses = %+v, want grpc endpoint", registered.TaggedAddresses)
	}
	if !strings.Contains(grpcAddr.Address, "isSecure=true") {
		t.Fatalf("registered grpc address = %q, want isSecure=true", grpcAddr.Address)
	}
	if len(registered.Checks) != 1 || registered.Checks[0].GRPC == "" || !registered.Checks[0].GRPCUseTLS {
		t.Fatalf("registered checks = %+v, want secure grpc health check", registered.Checks)
	}

	registryOptions := &appoptions.RegistryOptions{
		Address: fakeConsul.server.URL,
		Scheme:  "http",
	}
	conn, err := appclient.DialService(
		context.Background(),
		registryOptions,
		policy,
		"smoke-user-srv",
		rpcserver.WithConnectTimeout(2*time.Second),
		rpcserver.WithClientTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("DialService() error = %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	healthResp, err := healthClient.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health Check() error = %v", err)
	}
	if healthResp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("health status = %s, want SERVING", healthResp.GetStatus())
	}

	if err := smokeApp.StopContext(context.Background()); err != nil {
		t.Fatalf("StopContext() error = %v", err)
	}

	deregisteredID := fakeConsul.waitDeregister(t)
	if deregisteredID != registered.ID {
		t.Fatalf("deregistered id = %q, want %q", deregisteredID, registered.ID)
	}

	select {
	case runErr := <-runErrCh:
		if runErr != nil && !errors.Is(runErr, context.Canceled) {
			t.Fatalf("RunContext() error = %v, want nil or context canceled", runErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunContext() did not return after StopContext")
	}
}

type fakeConsulRegister struct {
	ID              string                        `json:"ID"`
	Name            string                        `json:"Name"`
	Meta            map[string]string             `json:"Meta"`
	Tags            []string                      `json:"Tags"`
	TaggedAddresses map[string]api.ServiceAddress `json:"TaggedAddresses"`
	Checks          []struct {
		GRPC       string `json:"GRPC"`
		GRPCUseTLS bool   `json:"GRPCUseTLS"`
		HTTP       string `json:"HTTP"`
		TCP        string `json:"TCP"`
	} `json:"Checks"`
}

type fakeConsulServer struct {
	server *httptest.Server

	mu       sync.Mutex
	index    uint64
	services map[string]fakeConsulRegister

	registerCh   chan fakeConsulRegister
	deregisterCh chan string
}

func newFakeConsulServer(t *testing.T) *fakeConsulServer {
	t.Helper()

	f := &fakeConsulServer{
		index:        1,
		services:     make(map[string]fakeConsulRegister),
		registerCh:   make(chan fakeConsulRegister, 1),
		deregisterCh: make(chan string, 1),
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeConsulServer) handle(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPut && r.URL.Path == "/v1/agent/service/register":
		var payload fakeConsulRegister
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.services[payload.ID] = payload
		f.index++
		f.mu.Unlock()
		select {
		case f.registerCh <- payload:
		default:
		}
		w.WriteHeader(http.StatusOK)
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/v1/agent/service/deregister/"):
		serviceID := path.Base(r.URL.Path)
		f.mu.Lock()
		delete(f.services, serviceID)
		f.index++
		f.mu.Unlock()
		select {
		case f.deregisterCh <- serviceID:
		default:
		}
		w.WriteHeader(http.StatusOK)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/health/service/"):
		serviceName := path.Base(r.URL.Path)
		entries, idx := f.serviceEntries(serviceName)
		w.Header().Set("X-Consul-Index", fmt.Sprintf("%d", idx))
		_ = json.NewEncoder(w).Encode(entries)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeConsulServer) serviceEntries(serviceName string) ([]*api.ServiceEntry, uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries := make([]*api.ServiceEntry, 0, len(f.services))
	for _, service := range f.services {
		if service.Name != serviceName {
			continue
		}
		entry := &api.ServiceEntry{
			Service: &api.AgentService{
				ID:              service.ID,
				Service:         service.Name,
				Meta:            service.Meta,
				Tags:            append([]string(nil), service.Tags...),
				TaggedAddresses: service.TaggedAddresses,
			},
		}
		if grpcAddr, ok := service.TaggedAddresses["grpc"]; ok {
			entry.Service.Address = grpcAddr.Address
			entry.Service.Port = grpcAddr.Port
		}
		entries = append(entries, entry)
	}
	return entries, f.index
}

func (f *fakeConsulServer) waitRegister(t *testing.T) fakeConsulRegister {
	t.Helper()

	select {
	case payload := <-f.registerCh:
		return payload
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for consul service registration")
		return fakeConsulRegister{}
	}
}

func (f *fakeConsulServer) waitDeregister(t *testing.T) string {
	t.Helper()

	select {
	case serviceID := <-f.deregisterCh:
		return serviceID
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for consul service deregistration")
		return ""
	}
}

func newSmokeSecurityPolicy(t *testing.T, serverName string) *rpcserver.SecurityPolicy {
	t.Helper()

	certPEM, keyPEM := newSmokeMutualTLSPEM(t, serverName)
	dir := t.TempDir()
	certFile := filepath.Join(dir, "internal.crt")
	keyFile := filepath.Join(dir, "internal.key")
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		t.Fatalf("write cert file failed: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatalf("write key file failed: %v", err)
	}
	return &rpcserver.SecurityPolicy{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     certFile,
		ServerName: serverName,
	}
}

func newSmokeMutualTLSPEM(t *testing.T, serverName string) ([]byte, []byte) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate private key failed: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: serverName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:              []string{serverName},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create certificate failed: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key failed: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	return certPEM, keyPEM
}
