package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"goshop/gmicro/logging"
	"goshop/gmicro/registry"

	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
)

type discoveryResolver struct {
	w  registry.Watcher
	cc resolver.ClientConn

	ctx    context.Context
	cancel context.CancelFunc

	insecure bool

	stateMu                  sync.Mutex
	emptyStateTimer          *time.Timer
	emptyStatePublished      bool
	closed                   bool
	emptySnapshotGracePeriod time.Duration
}

const defaultEmptySnapshotGracePeriod = 5 * time.Second

func (r *discoveryResolver) watch() {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}
		ins, err := r.w.Next()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			logging.ErrorContext(r.ctx, "resolver watch failed", slog.Any("err", err))
			time.Sleep(time.Second)
			continue
		}
		r.update(ins)
	}
}

func (r *discoveryResolver) update(ins []*registry.ServiceInstance) {
	addrs := make([]resolver.Address, 0)
	endpoints := make(map[string]struct{})
	for _, in := range ins {
		endpoint, err := ParseEndpoint(in.Endpoints, "grpc", !r.insecure)
		if err != nil {
			logging.ErrorContext(r.ctx, "resolver endpoint parse failed", slog.Any("err", err))
			continue
		}
		if endpoint == "" {
			continue
		}
		// filter redundant endpoints
		if _, ok := endpoints[endpoint]; ok {
			continue
		}
		endpoints[endpoint] = struct{}{}
		addr := resolver.Address{
			ServerName: resolverServerName(in.Metadata),
			Attributes: parseAttributes(in.Metadata),
			Addr:       endpoint,
		}
		addr.Attributes = addr.Attributes.WithValue("rawServiceInstance", in)
		addrs = append(addrs, addr)
	}
	if len(addrs) == 0 {
		r.deferEmptyState(ins)
		return
	}

	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.closed {
		return
	}
	if r.emptyStateTimer != nil {
		r.emptyStateTimer.Stop()
		r.emptyStateTimer = nil
	}
	r.emptyStatePublished = false
	r.publishStateLocked(resolver.State{Addresses: addrs})
	b, _ := json.Marshal(ins)
	logging.InfoContext(r.ctx, "resolver instances updated", slog.String("instances", string(b)))
}

func (r *discoveryResolver) deferEmptyState(ins []*registry.ServiceInstance) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.closed || r.emptyStatePublished || r.emptyStateTimer != nil {
		return
	}

	gracePeriod := r.emptySnapshotGracePeriod
	if gracePeriod <= 0 {
		gracePeriod = defaultEmptySnapshotGracePeriod
	}
	r.emptyStateTimer = time.AfterFunc(gracePeriod, r.publishEmptyState)
	logging.WarnContext(r.ctx, "resolver retained last state because no valid endpoint was found",
		slog.Duration("grace_period", gracePeriod),
		slog.Any("instances", ins),
	)
}

func (r *discoveryResolver) publishEmptyState() {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.closed || r.emptyStatePublished {
		return
	}
	r.emptyStateTimer = nil
	r.emptyStatePublished = true
	r.publishStateLocked(resolver.State{})
	logging.WarnContext(r.ctx, "resolver published empty state after grace period")
}

func (r *discoveryResolver) publishStateLocked(state resolver.State) {
	err := r.cc.UpdateState(state)
	if err != nil {
		logging.ErrorContext(r.ctx, "resolver state update failed", slog.Any("err", err))
	}
}

func (r *discoveryResolver) Close() {
	r.stateMu.Lock()
	if r.closed {
		r.stateMu.Unlock()
		return
	}
	r.closed = true
	if r.emptyStateTimer != nil {
		r.emptyStateTimer.Stop()
		r.emptyStateTimer = nil
	}
	r.stateMu.Unlock()

	if r.cancel != nil {
		r.cancel()
	}
	if r.w != nil {
		if err := r.w.Stop(); err != nil {
			logging.ErrorContext(r.ctx, "resolver watcher stop failed", slog.Any("err", err))
		}
	}
}

func (r *discoveryResolver) ResolveNow(options resolver.ResolveNowOptions) {}

func parseAttributes(md map[string]string) *attributes.Attributes {
	var a *attributes.Attributes
	for k, v := range md {
		if a == nil {
			a = attributes.New(k, v)
		} else {
			a = a.WithValue(k, v)
		}
	}
	return a
}

func resolverServerName(md map[string]string) string {
	if len(md) == 0 {
		return ""
	}
	if serverName := strings.TrimSpace(md["tls_server_name"]); serverName != "" {
		return serverName
	}
	if serverName := strings.TrimSpace(md["server_name"]); serverName != "" {
		return serverName
	}
	return ""
}

// NewEndpoint new an Endpoint URL.
func NewEndpoint(scheme, host string, isSecure bool) *url.URL {
	var query string
	if isSecure {
		query = "isSecure=true"
	}
	return &url.URL{Scheme: scheme, Host: host, RawQuery: query}
}

// ParseEndpoint parses an Endpoint URL.
func ParseEndpoint(endpoints []string, scheme string, isSecure bool) (string, error) {
	for _, e := range endpoints {
		u, err := url.Parse(e)
		if err != nil {
			return "", err
		}
		if u.Scheme == scheme {
			if IsSecure(u) == isSecure {
				return u.Host, nil
			}
		}
	}
	return "", nil
}

// IsSecure parses isSecure for Endpoint URL.
func IsSecure(u *url.URL) bool {
	ok, err := strconv.ParseBool(u.Query().Get("isSecure"))
	if err != nil {
		return false
	}
	return ok
}
