package resilience

import (
	"errors"
	"strings"

	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
	"github.com/prometheus/client_golang/prometheus"
)

const metricsNamespace = "dependency_resilience"

var (
	metricRequestsTotal      = registerCounterVec("requests_total", "External dependency operations by outcome.", []string{"dependency", "resource", "outcome"})
	metricDuration           = registerHistogramVec("duration_ms", "External dependency operation duration in milliseconds.", []string{"dependency", "resource", "outcome"}, []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000})
	metricInflight           = registerGaugeVec("inflight", "Current external dependency operations.", []string{"dependency", "resource"})
	metricFallbackTotal      = registerCounterVec("fallback_total", "External dependency fail-fast fallbacks by Sentinel block reason.", []string{"dependency", "resource", "reason"})
	metricCircuitTransitions = registerCounterVec("circuit_transitions_total", "External dependency circuit breaker state transitions.", []string{"dependency", "resource", "from", "to"})
	metricCircuitState       = registerGaugeVec("circuit_state", "External dependency circuit state: 0 closed, 1 half-open, 2 open.", []string{"dependency", "resource"})
	metricRecoveryTotal      = registerCounterVec("recovery_total", "External dependency circuit recoveries to closed state.", []string{"dependency", "resource"})
)

func registerCounterVec(name, help string, labels []string) *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: metricsNamespace, Name: name, Help: help}, labels)
	if err := prometheus.Register(vec); err != nil {
		var registered prometheus.AlreadyRegisteredError
		if errors.As(err, &registered) {
			if existing, ok := registered.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}
		return nil
	}
	return vec
}

func registerGaugeVec(name, help string, labels []string) *prometheus.GaugeVec {
	vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: metricsNamespace, Name: name, Help: help}, labels)
	if err := prometheus.Register(vec); err != nil {
		var registered prometheus.AlreadyRegisteredError
		if errors.As(err, &registered) {
			if existing, ok := registered.ExistingCollector.(*prometheus.GaugeVec); ok {
				return existing
			}
		}
		return nil
	}
	return vec
}

func registerHistogramVec(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: metricsNamespace, Name: name, Help: help, Buckets: buckets}, labels)
	if err := prometheus.Register(vec); err != nil {
		var registered prometheus.AlreadyRegisteredError
		if errors.As(err, &registered) {
			if existing, ok := registered.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing
			}
		}
		return nil
	}
	return vec
}

func count(vec *prometheus.CounterVec, labels ...string) {
	if vec != nil {
		vec.WithLabelValues(labels...).Inc()
	}
}
func addGauge(vec *prometheus.GaugeVec, value float64, labels ...string) {
	if vec != nil {
		vec.WithLabelValues(labels...).Add(value)
	}
}
func setGauge(vec *prometheus.GaugeVec, value float64, labels ...string) {
	if vec != nil {
		vec.WithLabelValues(labels...).Set(value)
	}
}
func observe(vec *prometheus.HistogramVec, value int64, labels ...string) {
	if vec != nil {
		vec.WithLabelValues(labels...).Observe(float64(value))
	}
}

type stateChangeListener struct{}

func (stateChangeListener) OnTransformToClosed(prev circuitbreaker.State, rule circuitbreaker.Rule) {
	recordCircuitTransition(prev, circuitbreaker.Closed, rule.Resource)
	dependency, resource := splitResource(rule.Resource)
	count(metricRecoveryTotal, dependency, resource)
}
func (stateChangeListener) OnTransformToOpen(prev circuitbreaker.State, rule circuitbreaker.Rule, _ interface{}) {
	recordCircuitTransition(prev, circuitbreaker.Open, rule.Resource)
}
func (stateChangeListener) OnTransformToHalfOpen(prev circuitbreaker.State, rule circuitbreaker.Rule) {
	recordCircuitTransition(prev, circuitbreaker.HalfOpen, rule.Resource)
}
func recordCircuitTransition(from, to circuitbreaker.State, sentinelResource string) {
	dependency, resource := splitResource(sentinelResource)
	count(metricCircuitTransitions, dependency, resource, stateName(from), stateName(to))
	setGauge(metricCircuitState, float64(to), dependency, resource)
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
