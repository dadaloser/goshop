package resilience

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"goshop/pkg/common/util/contextutil"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
	"github.com/alibaba/sentinel-golang/core/isolation"
)

var ErrBlocked = errors.New("dependency temporarily unavailable")
var listenerOnce sync.Once

type ErrorClassifier func(error) bool
type Guard struct {
	dependency string
	options    Options
	classifier ErrorClassifier
	configured sync.Map
}
type resourceConfiguration struct {
	once sync.Once
	err  error
}
type Call struct {
	guard    *Guard
	resource string
	ctx      context.Context
	cancel   context.CancelFunc
	entry    *base.SentinelEntry
	start    time.Time
	once     sync.Once
}
type BlockedError struct {
	Resource string
	Reason   string
}

func (e *BlockedError) Error() string {
	if e == nil {
		return "resilience: dependency temporarily unavailable"
	}
	return fmt.Sprintf("resilience: %s: %s", e.Resource, ErrBlocked)
}
func (e *BlockedError) Unwrap() error { return ErrBlocked }

func NewGuard(dependency string, options *Options, classifier ErrorClassifier) (*Guard, error) {
	if dependency == "" || strings.Contains(dependency, ":") {
		return nil, errors.New("resilience dependency must be non-empty and must not contain ':'")
	}
	if options == nil {
		options = NewOptions()
	}
	if errs := options.Validate(); len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	if classifier == nil {
		classifier = func(err error) bool { return err != nil }
	}
	listenerOnce.Do(func() { circuitbreaker.RegisterStateChangeListeners(stateChangeListener{}) })
	return &Guard{dependency: dependency, options: *options, classifier: classifier}, nil
}

func (g *Guard) Start(ctx context.Context, resource string) (*Call, error) {
	if g == nil {
		return nil, errors.New("resilience guard is required")
	}
	if resource == "" || strings.Contains(resource, ":") {
		return nil, errors.New("resilience resource must be non-empty and must not contain ':'")
	}
	if ctx == nil {
		ctx = contextutil.Root()
	}
	if err := g.configure(resource); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, g.options.Timeout)
	call := &Call{guard: g, resource: resource, ctx: ctx, cancel: cancel, start: time.Now()}
	addGauge(metricInflight, 1, g.dependency, resource)
	if !g.options.Enabled {
		return call, nil
	}
	entry, blockErr := sentinel.Entry(g.sentinelResource(resource), sentinel.WithResourceType(base.ResTypeCommon), sentinel.WithTrafficType(base.Outbound))
	if blockErr == nil {
		call.entry = entry
		return call, nil
	}
	reason := blockReason(blockErr.BlockType())
	addGauge(metricInflight, -1, g.dependency, resource)
	count(metricRequestsTotal, g.dependency, resource, "blocked")
	count(metricFallbackTotal, g.dependency, resource, reason)
	observe(metricDuration, 0, g.dependency, resource, "blocked")
	cancel()
	return nil, &BlockedError{Resource: g.sentinelResource(resource), Reason: reason}
}

func (c *Call) Context() context.Context {
	if c == nil {
		ctx, cancel := contextutil.NewOperation(time.Nanosecond)
		cancel()
		return ctx
	}
	return c.ctx
}
func (c *Call) Finish(err error) {
	if c == nil {
		return
	}
	c.once.Do(func() {
		if c.entry != nil {
			if err != nil && c.guard.classifier(err) {
				sentinel.TraceError(c.entry, err)
			}
			c.entry.Exit()
		}
		outcome := operationOutcome(err)
		addGauge(metricInflight, -1, c.guard.dependency, c.resource)
		count(metricRequestsTotal, c.guard.dependency, c.resource, outcome)
		observe(metricDuration, int64(time.Since(c.start)/time.Millisecond), c.guard.dependency, c.resource, outcome)
		c.cancel()
	})
}
func (g *Guard) Do(ctx context.Context, resource string, fn func(context.Context) error) error {
	if fn == nil {
		return errors.New("resilience operation is required")
	}
	call, err := g.Start(ctx, resource)
	if err != nil {
		return err
	}
	err = fn(call.Context())
	if err == nil && call.Context().Err() != nil {
		err = call.Context().Err()
	}
	call.Finish(err)
	return err
}
func (g *Guard) configure(resource string) error {
	if !g.options.Enabled {
		return nil
	}
	value, _ := g.configured.LoadOrStore(resource, &resourceConfiguration{})
	configuration := value.(*resourceConfiguration)
	configuration.once.Do(func() { configuration.err = g.loadRules(resource) })
	if configuration.err != nil {
		g.configured.Delete(resource)
	}
	return configuration.err
}
func (g *Guard) loadRules(resource string) error {
	name := g.sentinelResource(resource)
	if _, err := circuitbreaker.LoadRulesOfResource(name, []*circuitbreaker.Rule{{
		Resource:         name,
		Strategy:         circuitbreaker.ErrorRatio,
		RetryTimeoutMs:   uint32(g.options.RecoveryTimeout / time.Millisecond),
		MinRequestAmount: g.options.MinRequestAmount,
		StatIntervalMs:   uint32(g.options.StatInterval / time.Millisecond),
		Threshold:        g.options.ErrorRatio,
		ProbeNum:         1,
	}}); err != nil {
		return fmt.Errorf("load circuit breaker rule for %s: %w", name, err)
	}

	if _, err := isolation.LoadRulesOfResource(name, []*isolation.Rule{{
		Resource:   name,
		MetricType: isolation.Concurrency,
		Threshold:  g.options.MaxConcurrency,
	}}); err != nil {
		loadErr := fmt.Errorf("load isolation rule for %s: %w", name, err)
		if clearErr := circuitbreaker.ClearRulesOfResource(name); clearErr != nil {
			return errors.Join(loadErr, fmt.Errorf("clear circuit breaker rule for %s: %w", name, clearErr))
		}
		return loadErr
	}
	return nil
}
func (g *Guard) sentinelResource(resource string) string { return g.dependency + ":" + resource }
func operationOutcome(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "error"
	}
}
func blockReason(blockType base.BlockType) string {
	switch blockType {
	case base.BlockTypeIsolation:
		return "isolation"
	case base.BlockTypeCircuitBreaking:
		return "circuit_open"
	case base.BlockTypeFlow:
		return "flow"
	case base.BlockTypeSystemFlow:
		return "system"
	case base.BlockTypeHotSpotParamFlow:
		return "hotspot"
	default:
		return "unknown"
	}
}
