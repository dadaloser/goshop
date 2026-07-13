package resilience

import (
	"strings"

	"goshop/gmicro/core/metric"

	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
)

const metricsNamespace = "dependency_resilience"

var (
	metricRequestsTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: metricsNamespace,
		Name:      "requests_total",
		Help:      "External dependency operations by outcome.",
		Labels:    []string{"dependency", "resource", "outcome"},
	})
	metricDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: metricsNamespace,
		Name:      "duration_ms",
		Help:      "External dependency operation duration in milliseconds.",
		Labels:    []string{"dependency", "resource", "outcome"},
		Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000},
	})
	metricInflight = metric.NewGaugeVec(&metric.GaugeVecOpts{
		Namespace: metricsNamespace,
		Name:      "inflight",
		Help:      "Current external dependency operations.",
		Labels:    []string{"dependency", "resource"},
	})
	metricFallbackTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: metricsNamespace,
		Name:      "fallback_total",
		Help:      "External dependency fail-fast fallbacks by Sentinel block reason.",
		Labels:    []string{"dependency", "resource", "reason"},
	})
	metricCircuitTransitions = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: metricsNamespace,
		Name:      "circuit_transitions_total",
		Help:      "External dependency circuit breaker state transitions.",
		Labels:    []string{"dependency", "resource", "from", "to"},
	})
	metricCircuitState = metric.NewGaugeVec(&metric.GaugeVecOpts{
		Namespace: metricsNamespace,
		Name:      "circuit_state",
		Help:      "External dependency circuit state: 0 closed, 1 half-open, 2 open.",
		Labels:    []string{"dependency", "resource"},
	})
	metricRecoveryTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: metricsNamespace,
		Name:      "recovery_total",
		Help:      "External dependency circuit recoveries to closed state.",
		Labels:    []string{"dependency", "resource"},
	})
)

type stateChangeListener struct{}

func (stateChangeListener) OnTransformToClosed(prev circuitbreaker.State, rule circuitbreaker.Rule) {
	recordCircuitTransition(prev, circuitbreaker.Closed, rule.Resource)
	dependency, resource := splitResource(rule.Resource)
	metricRecoveryTotal.Inc(dependency, resource)
}

func (stateChangeListener) OnTransformToOpen(prev circuitbreaker.State, rule circuitbreaker.Rule, _ interface{}) {
	recordCircuitTransition(prev, circuitbreaker.Open, rule.Resource)
}

func (stateChangeListener) OnTransformToHalfOpen(prev circuitbreaker.State, rule circuitbreaker.Rule) {
	recordCircuitTransition(prev, circuitbreaker.HalfOpen, rule.Resource)
}

func recordCircuitTransition(from, to circuitbreaker.State, sentinelResource string) {
	dependency, resource := splitResource(sentinelResource)
	metricCircuitTransitions.Inc(dependency, resource, stateName(from), stateName(to))
	metricCircuitState.Set(float64(to), dependency, resource)
}

func splitResource(value string) (string, string) {
	dependency, resource, ok := strings.Cut(value, ":")
	if !ok {
		return "unknown", value
	}
	return dependency, resource
}

func stateName(state circuitbreaker.State) string {
	switch state {
	case circuitbreaker.Closed:
		return "closed"
	case circuitbreaker.HalfOpen:
		return "half_open"
	case circuitbreaker.Open:
		return "open"
	default:
		return "unknown"
	}
}
