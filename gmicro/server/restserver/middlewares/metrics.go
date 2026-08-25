package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsOptions configures HTTP metrics ownership for one REST server.
type MetricsOptions struct {
	Registerer prometheus.Registerer
	Namespace  string
}

type httpMetrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInflight *prometheus.GaugeVec
}

// NewMetrics creates an HTTP metrics middleware using the supplied registerer.
// A nil registerer uses Prometheus's process-wide default for compatibility.
func NewMetrics(service string, options MetricsOptions) (gin.HandlerFunc, error) {
	if options.Registerer == nil {
		options.Registerer = prometheus.DefaultRegisterer
	}
	if options.Namespace == "" {
		options.Namespace = "gmicro"
	}
	metrics := &httpMetrics{
		requestsTotal:    prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: options.Namespace, Subsystem: "http_server", Name: "requests_total", Help: "Total number of completed HTTP requests."}, []string{"service", "method", "route", "code"}),
		requestDuration:  prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: options.Namespace, Subsystem: "http_server", Name: "request_duration_seconds", Help: "HTTP request duration in seconds.", Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 15, 30}}, []string{"service", "method", "route"}),
		requestsInflight: prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: options.Namespace, Subsystem: "http_server", Name: "requests_inflight", Help: "Current number of in-flight HTTP requests."}, []string{"service"}),
	}
	var err error
	if metrics.requestsTotal, err = registerCounter(options.Registerer, metrics.requestsTotal); err != nil {
		return nil, err
	}
	if metrics.requestDuration, err = registerHistogram(options.Registerer, metrics.requestDuration); err != nil {
		return nil, err
	}
	if metrics.requestsInflight, err = registerGauge(options.Registerer, metrics.requestsInflight); err != nil {
		return nil, err
	}
	return metrics.handler(service), nil
}

func registerCounter(registerer prometheus.Registerer, collector *prometheus.CounterVec) (*prometheus.CounterVec, error) {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("register HTTP counter: %w", err)
	}
	return collector, nil
}

func registerHistogram(registerer prometheus.Registerer, collector *prometheus.HistogramVec) (*prometheus.HistogramVec, error) {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("register HTTP histogram: %w", err)
	}
	return collector, nil
}

func registerGauge(registerer prometheus.Registerer, collector *prometheus.GaugeVec) (*prometheus.GaugeVec, error) {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.GaugeVec); ok {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("register HTTP gauge: %w", err)
	}
	return collector, nil
}

func (m *httpMetrics) handler(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		m.requestsInflight.WithLabelValues(service).Inc()
		defer m.requestsInflight.WithLabelValues(service).Dec()
		defer func() {
			route := c.FullPath()
			if route == "" {
				route = "__unmatched__"
			}
			method := c.Request.Method
			if recovered := recover(); recovered != nil {
				m.requestsTotal.WithLabelValues(service, method, route, strconv.Itoa(http.StatusInternalServerError)).Inc()
				m.requestDuration.WithLabelValues(service, method, route).Observe(time.Since(started).Seconds())
				panic(recovered)
			}
			m.requestsTotal.WithLabelValues(service, method, route, strconv.Itoa(c.Writer.Status())).Inc()
			m.requestDuration.WithLabelValues(service, method, route).Observe(time.Since(started).Seconds())
		}()

		c.Next()
	}
}
