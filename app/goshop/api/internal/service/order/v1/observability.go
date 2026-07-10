package v1

import (
	"strings"
	"time"

	"goshop/app/pkg/code"
	"goshop/gmicro/core/metric"
	"goshop/pkg/errors"
)

const apiOrderMetricsNamespace = "goshop_api_order"

var (
	metricPayCallbackTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: apiOrderMetricsNamespace,
		Subsystem: "payment",
		Name:      "callback_total",
		Help:      "payment callback attempts grouped by target status and result.",
		Labels:    []string{"target_status", "result"},
	})

	metricPayCallbackDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: apiOrderMetricsNamespace,
		Subsystem: "payment",
		Name:      "callback_duration_ms",
		Help:      "payment callback duration in milliseconds.",
		Labels:    []string{"target_status", "result"},
		Buckets:   []float64{5, 10, 25, 50, 100, 250, 500, 1000, 3000, 5000},
	})
)

func observePayCallback(targetStatus, result string, startedAt time.Time) {
	targetStatus = normalizePayCallbackTargetStatus(targetStatus)
	result = normalizePayCallbackResult(result)
	metricPayCallbackTotal.Inc(targetStatus, result)
	metricPayCallbackDuration.Observe(int64(time.Since(startedAt)/time.Millisecond), targetStatus, result)
}

func normalizePayCallbackTargetStatus(status string) string {
	switch strings.TrimSpace(status) {
	case orderStatusTradeSuccess:
		return "trade_success"
	case orderStatusTradeClosed:
		return "trade_closed"
	default:
		return "unknown"
	}
}

func payCallbackMetricResult(err error) string {
	if err == nil {
		return "success"
	}
	switch {
	case errors.IsCode(err, code.ErrOrderStatusInvalid), errors.IsCode(err, code.ErrOrderNotFound):
		return "rejected"
	case errors.IsCode(err, code.ErrConnectGRPC):
		return "dependency_error"
	default:
		return "failed"
	}
}

func normalizePayCallbackResult(result string) string {
	switch strings.TrimSpace(result) {
	case "success", "rejected", "dependency_error", "failed":
		return result
	default:
		return "failed"
	}
}
