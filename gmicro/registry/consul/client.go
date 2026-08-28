package consul

import (
	"context"
	"fmt"
	"goshop/gmicro/contextutil"
	"goshop/gmicro/logging"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"goshop/gmicro/registry"

	"github.com/hashicorp/consul/api"
)

// 检查心跳失败次数
const heartbeatFailureThreshold = 3
const consulOperationTimeout = 10 * time.Second

// Client is consul client config
type Client struct {
	client *api.Client

	heartbeatMu      sync.Mutex
	heartbeatCancels map[string]context.CancelFunc
	// cancel stops all heartbeats. It is retained for component-level cleanup.
	cancel context.CancelFunc

	// resolve service entry endpoints
	resolver ServiceResolver
	// healthcheck time interval in seconds
	healthcheckInterval int
	// heartbeat enable heartbeat
	heartbeat bool
	// heartbeatTimeout limits each TTL update request.
	heartbeatTimeout time.Duration
	// deregisterCriticalServiceAfter time interval in seconds
	deregisterCriticalServiceAfter int
	// serviceChecks  user custom checks
	serviceChecks api.AgentServiceChecks
	// httpHealthCheckPath is used for http/https endpoint checks.
	httpHealthCheckPath string
}

// NewClient creates consul client
func NewClient(cli *api.Client) *Client {
	c := &Client{
		client:                         cli,
		resolver:                       defaultResolver,
		healthcheckInterval:            10,
		heartbeat:                      false,
		heartbeatTimeout:               5 * time.Second,
		deregisterCriticalServiceAfter: 600,
		httpHealthCheckPath:            "/readyz",
		heartbeatCancels:               make(map[string]context.CancelFunc),
	}
	c.cancel = c.stopAllHeartbeats
	return c
}

func defaultResolver(_ context.Context, entries []*api.ServiceEntry) []*registry.ServiceInstance {
	services := make([]*registry.ServiceInstance, 0, len(entries))
	for _, entry := range entries {
		var version string
		for _, tag := range entry.Service.Tags {
			ss := strings.SplitN(tag, "=", 2)
			if len(ss) == 2 && ss[0] == "version" {
				version = ss[1]
			}
		}
		endpoints := make([]string, 0)
		for scheme, addr := range entry.Service.TaggedAddresses {
			if scheme == "lan_ipv4" || scheme == "wan_ipv4" || scheme == "lan_ipv6" || scheme == "wan_ipv6" {
				continue
			}
			endpoints = append(endpoints, addr.Address)
		}
		if len(endpoints) == 0 && entry.Service.Address != "" && entry.Service.Port != 0 {
			endpoints = append(endpoints, fmt.Sprintf("http://%s:%d", entry.Service.Address, entry.Service.Port))
		}
		services = append(services, &registry.ServiceInstance{
			ID:        entry.Service.ID,
			Name:      entry.Service.Service,
			Metadata:  entry.Service.Meta,
			Version:   version,
			Endpoints: endpoints,
		})
	}

	return services
}

// ServiceResolver is used to resolve service endpoints
type ServiceResolver func(ctx context.Context, entries []*api.ServiceEntry) []*registry.ServiceInstance

// Service get services from consul
func (c *Client) Service(ctx context.Context, service string, index uint64, passingOnly bool) ([]*registry.ServiceInstance, uint64, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = contextutil.NewOperation(time.Minute)
		defer cancel()
	}
	opts := &api.QueryOptions{
		WaitIndex: index,
		WaitTime:  time.Second * 55,
	}
	opts = opts.WithContext(ctx)
	entries, meta, err := c.client.Health().Service(service, "", passingOnly, opts)
	if err != nil {
		return nil, 0, err
	}
	return c.resolver(ctx, entries), meta.LastIndex, nil
}

// Register register service instance to consul
func (c *Client) Register(ctx context.Context, svc *registry.ServiceInstance, enableHealthCheck bool) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = contextutil.NewOperation(consulOperationTimeout)
		defer cancel()
	}
	addresses := make(map[string]api.ServiceAddress, len(svc.Endpoints))
	checkAddresses := make([]string, 0, len(svc.Endpoints))
	checks := make(api.AgentServiceChecks, 0, len(svc.Endpoints))
	var healthCheckEndpoint *url.URL
	if svc.HealthCheckEndpoint != "" {
		parsed, err := url.Parse(svc.HealthCheckEndpoint)
		if err != nil {
			return fmt.Errorf("parse health check endpoint: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("invalid health check endpoint %q: scheme must be http or https", svc.HealthCheckEndpoint)
		}
		if _, _, err := parseEndpointAddress(parsed); err != nil {
			return err
		}
		healthCheckEndpoint = parsed
	}
	for _, endpoint := range svc.Endpoints {
		raw, err := url.Parse(endpoint)
		if err != nil {
			return err
		}
		addr, port, err := parseEndpointAddress(raw)
		if err != nil {
			return err
		}
		checkAddress := net.JoinHostPort(addr, strconv.FormatUint(uint64(port), 10))

		checkAddresses = append(checkAddresses, checkAddress)
		addresses[raw.Scheme] = api.ServiceAddress{Address: endpoint, Port: int(port)}
		if enableHealthCheck {
			switch raw.Scheme {
			case "http", "https":
				if healthCheckEndpoint != nil {
					// The dedicated management endpoint is the single source of
					// HTTP readiness; do not probe the business traffic port.
					continue
				}
				check := &api.AgentServiceCheck{
					Interval:                       fmt.Sprintf("%ds", c.healthcheckInterval),
					DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", c.deregisterCriticalServiceAfter),
					Timeout:                        "5s",
				}
				check.HTTP = c.healthCheckURL(raw)
				checks = append(checks, check)
			case "grpc":
				if endpointIsSecure(raw) && c.heartbeat {
					// Consul active gRPC TLS checks cannot complete a strict mTLS
					// handshake for internal services, so secure gRPC endpoints rely
					// on the agent-updated TTL heartbeat as the source of truth.
					break
				}
				check := &api.AgentServiceCheck{
					Interval:                       fmt.Sprintf("%ds", c.healthcheckInterval),
					DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", c.deregisterCriticalServiceAfter),
					Timeout:                        "5s",
				}
				check.GRPC = checkAddress
				check.GRPCUseTLS = endpointIsSecure(raw)
				checks = append(checks, check)
			default:
				check := &api.AgentServiceCheck{
					Interval:                       fmt.Sprintf("%ds", c.healthcheckInterval),
					DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", c.deregisterCriticalServiceAfter),
					Timeout:                        "5s",
				}
				check.TCP = checkAddress
				checks = append(checks, check)
			}
		}
	}
	if enableHealthCheck && healthCheckEndpoint != nil {
		checks = append(checks, &api.AgentServiceCheck{
			Interval:                       fmt.Sprintf("%ds", c.healthcheckInterval),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", c.deregisterCriticalServiceAfter),
			Timeout:                        "5s",
			HTTP:                           c.healthCheckURL(healthCheckEndpoint),
		})
	}
	asr := &api.AgentServiceRegistration{
		ID:              svc.ID,
		Name:            svc.Name,
		Meta:            svc.Metadata,
		Tags:            []string{fmt.Sprintf("version=%s", svc.Version)},
		TaggedAddresses: addresses,
	}
	if len(checkAddresses) > 0 {
		host, portRaw, _ := net.SplitHostPort(checkAddresses[0])
		port, _ := strconv.ParseInt(portRaw, 10, 32)
		asr.Address = host
		asr.Port = int(port)
	}
	if enableHealthCheck {
		asr.Checks = append(asr.Checks, checks...)
	}
	if c.heartbeat {
		asr.Checks = append(asr.Checks, &api.AgentServiceCheck{
			CheckID:                        "service:" + svc.ID,
			TTL:                            fmt.Sprintf("%ds", c.healthcheckInterval*2),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", c.deregisterCriticalServiceAfter),
		})
	}

	// custom checks
	asr.Checks = append(asr.Checks, c.serviceChecks...)

	err := c.client.Agent().ServiceRegisterOpts(asr, api.ServiceRegisterOpts{}.WithContext(ctx))
	if err != nil {
		return err
	}
	if c.heartbeat {
		heartbeatCtx := c.startHeartbeat(svc.ID)
		go func(ctx context.Context) {
			failures := 0
			updateTTL := func() {
				heartbeatCtx, cancel := context.WithTimeout(ctx, c.heartbeatTimeout)
				heartbeatErr := c.client.Agent().UpdateTTLOpts(
					"service:"+svc.ID,
					"pass",
					"pass",
					new(api.QueryOptions).WithContext(heartbeatCtx),
				)
				cancel()
				if heartbeatErr != nil {
					failures++
					logging.Error("consul ttl heartbeat update failed", slog.Any("err", heartbeatErr))
					if failures >= heartbeatFailureThreshold {
						//失败重新注册
						registerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
						registerErr := c.client.Agent().ServiceRegisterOpts(
							asr,
							api.ServiceRegisterOpts{}.WithContext(registerCtx),
						)
						cancel()
						if registerErr != nil {
							logging.Error("consul service re-register failed", slog.Any("err", registerErr))
							return
						}
						logging.Info("consul service re-register succeeded", slog.String("service_id", svc.ID))
						failures = 0
					}
					return
				}
				failures = 0
			}

			initialDelay := time.Second
			if c.healthcheckInterval < 1 {
				initialDelay = 0
			}
			timer := time.NewTimer(initialDelay)
			defer timer.Stop()
			select {
			case <-timer.C:
				updateTTL()
			case <-ctx.Done():
				return
			}
			tickerInterval := time.Second * time.Duration(c.healthcheckInterval)
			if tickerInterval <= 0 {
				tickerInterval = time.Second
			}
			ticker := time.NewTicker(tickerInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					updateTTL()
				case <-ctx.Done():
					return
				}
			}
		}(heartbeatCtx)
	}
	return nil
}

func (c *Client) startHeartbeat(serviceID string) context.Context {
	ctx, cancel := contextutil.NewProcess()

	c.heartbeatMu.Lock()
	previous := c.heartbeatCancels[serviceID]
	c.heartbeatCancels[serviceID] = cancel
	c.heartbeatMu.Unlock()

	if previous != nil {
		previous()
	}
	return ctx
}

func (c *Client) stopHeartbeat(serviceID string) {
	c.heartbeatMu.Lock()
	cancel := c.heartbeatCancels[serviceID]
	delete(c.heartbeatCancels, serviceID)
	c.heartbeatMu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (c *Client) stopAllHeartbeats() {
	c.heartbeatMu.Lock()
	cancels := c.heartbeatCancels
	c.heartbeatCancels = make(map[string]context.CancelFunc)
	c.heartbeatMu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}

func parseEndpointAddress(endpoint *url.URL) (string, uint16, error) {
	if endpoint.Scheme == "" {
		return "", 0, fmt.Errorf("invalid endpoint %q: missing scheme", endpoint.String())
	}
	addr := endpoint.Hostname()
	if addr == "" {
		return "", 0, fmt.Errorf("invalid endpoint %q: missing host", endpoint.String())
	}
	portRaw := endpoint.Port()
	if portRaw == "" {
		return "", 0, fmt.Errorf("invalid endpoint %q: missing port", endpoint.String())
	}
	port, err := strconv.ParseUint(portRaw, 10, 16)
	if err != nil {
		return "", 0, fmt.Errorf("invalid endpoint %q: invalid port: %w", endpoint.String(), err)
	}
	return addr, uint16(port), nil
}

func endpointIsSecure(endpoint *url.URL) bool {
	ok, err := strconv.ParseBool(endpoint.Query().Get("isSecure"))
	if err != nil {
		return false
	}
	return ok
}

func (c *Client) healthCheckURL(endpoint *url.URL) string {
	healthURL := *endpoint
	healthURL.Path = normalizeHTTPHealthCheckPath(c.httpHealthCheckPath)
	healthURL.RawPath = ""
	healthURL.RawQuery = ""
	healthURL.Fragment = ""
	return healthURL.String()
}

func normalizeHTTPHealthCheckPath(path string) string {
	if path == "" {
		return "/readyz"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

// Deregister deregister service by service ID
func (c *Client) Deregister(ctx context.Context, serviceID string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = contextutil.NewOperation(consulOperationTimeout)
		defer cancel()
	}
	if err := c.client.Agent().ServiceDeregisterOpts(serviceID, new(api.QueryOptions).WithContext(ctx)); err != nil {
		return err
	}
	c.stopHeartbeat(serviceID)
	return nil
}
