package resilience

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
	"github.com/alibaba/sentinel-golang/core/isolation"
)

// ErrBlocked identifies an operation rejected by dependency isolation or circuit breaking.
var ErrBlocked = errors.New("dependency temporarily unavailable")

var listenerOnce sync.Once

// ErrorClassifier reports whether an operation error should contribute to the circuit error ratio.
type ErrorClassifier func(error) bool

// Guard applies a dependency policy to low-cardinality resource names.
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

// Call represents one admitted dependency operation and must be finished exactly once.
type Call struct {
	guard    *Guard
	resource string
	ctx      context.Context
	cancel   context.CancelFunc
	entry    *base.SentinelEntry
	start    time.Time
	once     sync.Once
}

// BlockedError carries the protected resource while remaining matchable with ErrBlocked.
type BlockedError struct {
	Resource string
	Reason   string
}

// Error returns a stable dependency-unavailable message with the Sentinel resource name.
func (e *BlockedError) Error() string {
	if e == nil {
		return "resilience: dependency temporarily unavailable"
	}
	return fmt.Sprintf("resilience: %s: %s", e.Resource, ErrBlocked)
}

// Unwrap allows callers to detect blocked operations with errors.Is.
func (e *BlockedError) Unwrap() error {
	return ErrBlocked
}

// NewGuard validates and copies options for a dependency such as grpc, redis, or mysql.
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

	listenerOnce.Do(func() {
		circuitbreaker.RegisterStateChangeListeners(stateChangeListener{})
	})
	return &Guard{
		dependency: dependency,
		options:    *options,
		classifier: classifier,
	}, nil
}

// Start admits one operation, applies its timeout, and returns ErrBlocked on fallback.
func (g *Guard) Start(ctx context.Context, resource string) (*Call, error) {
	if g == nil {
		return nil, errors.New("resilience guard is required")
	}
	if resource == "" || strings.Contains(resource, ":") {
		return nil, errors.New("resilience resource must be non-empty and must not contain ':'")
	}
	if ctx == nil {
		ctx = context.TODO()
	}
	if err := g.configure(resource); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, g.options.Timeout)
	call := &Call{
		guard:    g,
		resource: resource,
		ctx:      ctx,
		cancel:   cancel,
		start:    time.Now(),
	}
	metricInflight.Inc(g.dependency, resource)

	if !g.options.Enabled {
		return call, nil
	}

	entry, blockErr := sentinel.Entry(
		g.sentinelResource(resource),
		sentinel.WithResourceType(base.ResTypeCommon),
		sentinel.WithTrafficType(base.Outbound),
	)
	if blockErr == nil {
		call.entry = entry
		return call, nil
	}

	reason := blockReason(blockErr.BlockType())
	metricInflight.Add(-1, g.dependency, resource)
	metricRequestsTotal.Inc(g.dependency, resource, "blocked")
	metricFallbackTotal.Inc(g.dependency, resource, reason)
	metricDuration.Observe(0, g.dependency, resource, "blocked")
	cancel()
	return nil, &BlockedError{
		Resource: g.sentinelResource(resource),
		Reason:   reason,
	}
}

// Context returns the timeout-bounded context for the dependency operation.
func (c *Call) Context() context.Context {
	if c == nil {
		return context.TODO()
	}
	return c.ctx
}

// Finish records the result, updates Sentinel statistics, and releases isolation capacity.
func (c *Call) Finish(err error) {
	if c == nil {
		return
	}
	c.once.Do(func() {
		if c.entry != nil {
			if err != nil && c.guard.classifier != nil && c.guard.classifier(err) {
				sentinel.TraceError(c.entry, err)
			}
			c.entry.Exit()
		}

		outcome := operationOutcome(err)
		metricInflight.Add(-1, c.guard.dependency, c.resource)
		metricRequestsTotal.Inc(c.guard.dependency, c.resource, outcome)
		metricDuration.Observe(
			int64(time.Since(c.start)/time.Millisecond),
			c.guard.dependency,
			c.resource,
			outcome,
		)
		c.cancel()
	})
}

// Do executes fn under the guard and returns a timeout if fn outlives the operation context.
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
	configuration.once.Do(func() {
		configuration.err = g.loadRules(resource)
	})
	if configuration.err != nil {
		g.configured.Delete(resource)
	}
	return configuration.err
}

func (g *Guard) loadRules(resource string) error {
	sentinelResource := g.sentinelResource(resource)
	_, err := circuitbreaker.LoadRulesOfResource(sentinelResource, []*circuitbreaker.Rule{
		{
			Resource:         sentinelResource,
			Strategy:         circuitbreaker.ErrorRatio,
			RetryTimeoutMs:   durationMillis(g.options.RecoveryTimeout),
			MinRequestAmount: g.options.MinRequestAmount,
			StatIntervalMs:   durationMillis(g.options.StatInterval),
			Threshold:        g.options.ErrorRatio,
			ProbeNum:         1,
		},
	})
	if err != nil {
		return fmt.Errorf("load circuit breaker rule for %s: %w", sentinelResource, err)
	}
	_, err = isolation.LoadRulesOfResource(sentinelResource, []*isolation.Rule{
		{
			Resource:   sentinelResource,
			MetricType: isolation.Concurrency,
			Threshold:  g.options.MaxConcurrency,
		},
	})
	if err != nil {
		loadErr := fmt.Errorf("load isolation rule for %s: %w", sentinelResource, err)
		if clearErr := circuitbreaker.ClearRulesOfResource(sentinelResource); clearErr != nil {
			return errors.Join(loadErr, fmt.Errorf("clear circuit breaker rule for %s: %w", sentinelResource, clearErr))
		}
		return loadErr
	}
	return nil
}

func (g *Guard) sentinelResource(resource string) string {
	return g.dependency + ":" + resource
}

func durationMillis(value time.Duration) uint32 {
	return uint32(value / time.Millisecond)
}

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
