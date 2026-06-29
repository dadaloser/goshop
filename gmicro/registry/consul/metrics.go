package consul

import "goshop/gmicro/core/metric"

const registryNamespace = "consul_registry"

var (
	metricConsulWatchErrors = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: registryNamespace,
		Subsystem: "watch",
		Name:      "errors_total",
		Help:      "consul service watch errors.",
		Labels:    []string{"service", "phase"},
	})

	metricConsulResolverLifecycle = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: registryNamespace,
		Subsystem: "watch",
		Name:      "resolver_lifecycle_total",
		Help:      "consul resolver lifecycle events.",
		Labels:    []string{"service", "event"},
	})
)
