package service

import (
	"strings"
	"time"

	"goshop/gmicro/core/metric"
)

const orderMetricsNamespace = "order_service"

var (
	metricOrderTransitionTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: orderMetricsNamespace,
		Subsystem: "state",
		Name:      "transition_total",
		Help:      "order state transitions grouped by trigger and result.",
		Labels:    []string{"trigger", "transition", "result"},
	})

	metricOrderTransitionDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: orderMetricsNamespace,
		Subsystem: "state",
		Name:      "transition_duration_ms",
		Help:      "order state transition duration in milliseconds.",
		Labels:    []string{"trigger", "transition", "result"},
		Buckets:   []float64{5, 10, 25, 50, 100, 250, 500, 1000, 3000, 5000},
	})

	metricOrderLifecycleCandidatesTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: orderMetricsNamespace,
		Subsystem: "lifecycle",
		Name:      "candidates_total",
		Help:      "number of lifecycle worker candidate orders discovered per operation.",
		Labels:    []string{"operation"},
	})

	metricOrderLifecycleProcessedTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: orderMetricsNamespace,
		Subsystem: "lifecycle",
		Name:      "processed_total",
		Help:      "number of lifecycle worker processed orders grouped by operation and result.",
		Labels:    []string{"operation", "result"},
	})

	metricOrderLifecycleSweepDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: orderMetricsNamespace,
		Subsystem: "lifecycle",
		Name:      "sweep_duration_ms",
		Help:      "lifecycle worker sweep duration in milliseconds.",
		Labels:    []string{"operation", "result"},
		Buckets:   []float64{5, 10, 25, 50, 100, 250, 500, 1000, 3000, 5000, 10000},
	})
)

func observeOrderTransition(trigger, transition, result string, startedAt time.Time) {
	trigger = normalizeTransitionTrigger(trigger)
	transition = normalizeTransitionName(transition)
	result = normalizeMetricResult(result)
	metricOrderTransitionTotal.Inc(trigger, transition, result)
	metricOrderTransitionDuration.Observe(durationMillis(startedAt), trigger, transition, result)
}

func observeLifecycleSweep(operation, result string, startedAt time.Time) {
	operation = normalizeLifecycleOperation(operation)
	result = normalizeMetricResult(result)
	metricOrderLifecycleSweepDuration.Observe(durationMillis(startedAt), operation, result)
}

func normalizeTransitionTrigger(source string) string {
	switch strings.TrimSpace(source) {
	case "", "order.update":
		return "update"
	case "order.create":
		return "create"
	case "order.payment":
		return "payment"
	case "order.timeout_worker":
		return "timeout_worker"
	case "order.finish_worker":
		return "finish_worker"
	case "order.close":
		return "close"
	case "order.finish":
		return "finish"
	default:
		return "custom"
	}
}

func transitionMetricName(fromStatus, toStatus string) string {
	fromStatus = strings.TrimSpace(fromStatus)
	toStatus = strings.TrimSpace(toStatus)

	switch {
	case fromStatus == "" && toStatus == OrderStatusWaitBuyerPay:
		return "create"
	case fromStatus == toStatus:
		return "noop"
	case toStatus == OrderStatusTradeSuccess:
		return "pay_success"
	case toStatus == OrderStatusTradeClosed:
		return "close"
	case toStatus == OrderStatusTradeFinished:
		return "finish"
	default:
		return "change"
	}
}

func normalizeTransitionName(name string) string {
	switch strings.TrimSpace(name) {
	case "create", "noop", "pay_success", "close", "finish", "change":
		return name
	default:
		return "change"
	}
}

func normalizeLifecycleOperation(operation string) string {
	switch strings.TrimSpace(operation) {
	case "auto_close", "auto_finish":
		return operation
	default:
		return "unknown"
	}
}

func normalizeMetricResult(result string) string {
	switch strings.TrimSpace(result) {
	case "success", "failed", "invalid", "noop":
		return result
	default:
		return "failed"
	}
}

func durationMillis(startedAt time.Time) int64 {
	if startedAt.IsZero() {
		return 0
	}
	return int64(time.Since(startedAt) / time.Millisecond)
}
