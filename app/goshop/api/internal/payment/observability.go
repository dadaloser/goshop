package payment

import "goshop/gmicro/core/metric"

const paymentMetricsNamespace = "goshop_api"

var (
	metricPaymentRefundJobsTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: paymentMetricsNamespace,
		Subsystem: "payment_worker",
		Name:      "refund_jobs_total",
		Help:      "Refund worker job outcomes grouped by result.",
		Labels:    []string{"result"},
	})

	metricPaymentReconciliationRunsTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: paymentMetricsNamespace,
		Subsystem: "payment_worker",
		Name:      "reconciliation_runs_total",
		Help:      "Payment reconciliation windows grouped by result.",
		Labels:    []string{"result"},
	})

	metricPaymentReconciliationMismatchCount = metric.NewGaugeVec(&metric.GaugeVecOpts{
		Namespace: paymentMetricsNamespace,
		Subsystem: "payment_worker",
		Name:      "reconciliation_mismatch_count",
		Help:      "Latest observed payment reconciliation mismatch count by provider.",
		Labels:    []string{"provider"},
	})

	metricPaymentCallbackHTTP = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: paymentMetricsNamespace,
		Subsystem: "payment_callback",
		Name:      "http_total",
		Help:      "Payment callback HTTP outcomes grouped by result.",
		Labels:    []string{"result"},
	})
)
