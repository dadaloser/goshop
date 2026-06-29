package consul

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"goshop/gmicro/registry"
	"goshop/pkg/log"

	"github.com/hashicorp/consul/api"
)

// 检查心跳失败次数
const heartbeatFailureThreshold = 3

// Client is consul client config
type Client struct {
	cli    *api.Client
	ctx    context.Context
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
		cli:                            cli,
		resolver:                       defaultResolver,
		healthcheckInterval:            10,
		heartbeat:                      true,
		heartbeatTimeout:               5 * time.Second,
		deregisterCriticalServiceAfter: 600,
		httpHealthCheckPath:            "/readyz",
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
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
	opts := &api.QueryOptions{
		WaitIndex: index,
		WaitTime:  time.Second * 55,
	}
	opts = opts.WithContext(ctx)
	entries, meta, err := c.cli.Health().Service(service, "", passingOnly, opts)
	if err != nil {
		return nil, 0, err
	}
	return c.resolver(ctx, entries), meta.LastIndex, nil
}

// Register register service instance to consul
func (c *Client) Register(ctx context.Context, svc *registry.ServiceInstance, enableHealthCheck bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	addresses := make(map[string]api.ServiceAddress, len(svc.Endpoints))
	checkAddresses := make([]string, 0, len(svc.Endpoints))
	checks := make(api.AgentServiceChecks, 0, len(svc.Endpoints))
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
			check := &api.AgentServiceCheck{
				Interval:                       fmt.Sprintf("%ds", c.healthcheckInterval),
				DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", c.deregisterCriticalServiceAfter),
				Timeout:                        "5s",
			}
			switch raw.Scheme {
			case "http", "https":
				check.HTTP = c.healthCheckURL(raw)
			default:
				check.TCP = checkAddress
			}
			checks = append(checks, check)
		}
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

	err := c.cli.Agent().ServiceRegisterOpts(asr, api.ServiceRegisterOpts{}.WithContext(ctx))
	if err != nil {
		return err
	}
	if c.heartbeat {
		go func() {
			failures := 0
			updateTTL := func() {
				heartbeatCtx, cancel := context.WithTimeout(c.ctx, c.heartbeatTimeout)
				heartbeatErr := c.cli.Agent().UpdateTTLOpts(
					"service:"+svc.ID,
					"pass",
					"pass",
					new(api.QueryOptions).WithContext(heartbeatCtx),
				)
				cancel()
				if heartbeatErr != nil {
					failures++
					log.Errorf("[Consul] update ttl heartbeat to consul failed: %v", heartbeatErr)
					if failures >= heartbeatFailureThreshold {
						//失败重新注册
						registerCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
						registerErr := c.cli.Agent().ServiceRegisterOpts(
							asr,
							api.ServiceRegisterOpts{}.WithContext(registerCtx),
						)
						cancel()
						if registerErr != nil {
							log.Errorf("[Consul] re-register service failed: %v", registerErr)
							return
						}
						log.Infof("[Consul] re-register service success: %s", svc.ID)
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
			case <-c.ctx.Done():
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
				case <-c.ctx.Done():
					return
				}
			}
		}()
	}
	return nil
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
		ctx = context.Background()
	}
	c.cancel()
	return c.cli.Agent().ServiceDeregisterOpts(serviceID, new(api.QueryOptions).WithContext(ctx))
}
