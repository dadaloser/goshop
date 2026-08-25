package middlewares

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gmicro",
		Subsystem: "http_server",
		Name:      "requests_total",
		Help:      "Total number of completed HTTP requests.",
	}, []string{"service", "method", "route", "code"})
	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "gmicro",
		Subsystem: "http_server",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 15, 30},
	}, []string{"service", "method", "route"})
	httpRequestsInflight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gmicro",
		Subsystem: "http_server",
		Name:      "requests_inflight",
		Help:      "Current number of in-flight HTTP requests.",
	}, []string{"service"})
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, httpRequestsInflight)
}

// Metrics records bounded-cardinality request metrics for one HTTP service.
func Metrics(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		httpRequestsInflight.WithLabelValues(service).Inc()
		defer httpRequestsInflight.WithLabelValues(service).Dec()
		defer func() {
			route := c.FullPath()
			if route == "" {
				route = "__unmatched__"
			}
			method := c.Request.Method
			if recovered := recover(); recovered != nil {
				httpRequestsTotal.WithLabelValues(service, method, route, strconv.Itoa(http.StatusInternalServerError)).Inc()
				httpRequestDuration.WithLabelValues(service, method, route).Observe(time.Since(started).Seconds())
				panic(recovered)
			}
			httpRequestsTotal.WithLabelValues(service, method, route, strconv.Itoa(c.Writer.Status())).Inc()
			httpRequestDuration.WithLabelValues(service, method, route).Observe(time.Since(started).Seconds())
		}()

		c.Next()
	}
}
