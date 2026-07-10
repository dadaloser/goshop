package metric

import (
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
)

func TestNewCounterVecReusesExistingCollector(t *testing.T) {
	reg := swapDefaultRegistry(t)

	counterOne := NewCounterVec(&CounterVecOpts{
		Namespace: "test_metric",
		Subsystem: "counter",
		Name:      "duplicate_total",
		Help:      "duplicate counter",
		Labels:    []string{"kind"},
	})
	counterTwo := NewCounterVec(&CounterVecOpts{
		Namespace: "test_metric",
		Subsystem: "counter",
		Name:      "duplicate_total",
		Help:      "duplicate counter",
		Labels:    []string{"kind"},
	})

	counterOne.Inc("ok")
	counterTwo.Inc("ok")

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	if len(families) != 1 {
		t.Fatalf("Gather() families = %d, want 1", len(families))
	}
	if got := families[0].GetMetric()[0].GetCounter().GetValue(); got != 2 {
		t.Fatalf("counter value = %v, want 2", got)
	}
}

func TestNewGaugeVecReusesExistingCollector(t *testing.T) {
	reg := swapDefaultRegistry(t)

	gaugeOne := NewGaugeVec(&GaugeVecOpts{
		Namespace: "test_metric",
		Subsystem: "gauge",
		Name:      "duplicate",
		Help:      "duplicate gauge",
		Labels:    []string{"kind"},
	})
	gaugeTwo := NewGaugeVec(&GaugeVecOpts{
		Namespace: "test_metric",
		Subsystem: "gauge",
		Name:      "duplicate",
		Help:      "duplicate gauge",
		Labels:    []string{"kind"},
	})

	gaugeOne.Set(1, "ok")
	gaugeTwo.Add(2, "ok")

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	if len(families) != 1 {
		t.Fatalf("Gather() families = %d, want 1", len(families))
	}
	if got := families[0].GetMetric()[0].GetGauge().GetValue(); got != 3 {
		t.Fatalf("gauge value = %v, want 3", got)
	}
}

func TestNewHistogramVecReusesExistingCollector(t *testing.T) {
	reg := swapDefaultRegistry(t)

	histogramOne := NewHistogramVec(&HistogramVecOpts{
		Namespace: "test_metric",
		Subsystem: "histogram",
		Name:      "duplicate_ms",
		Help:      "duplicate histogram",
		Labels:    []string{"kind"},
		Buckets:   []float64{10, 20, 50},
	})
	histogramTwo := NewHistogramVec(&HistogramVecOpts{
		Namespace: "test_metric",
		Subsystem: "histogram",
		Name:      "duplicate_ms",
		Help:      "duplicate histogram",
		Labels:    []string{"kind"},
		Buckets:   []float64{10, 20, 50},
	})

	histogramOne.Observe(10, "ok")
	histogramTwo.Observe(20, "ok")

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	if len(families) != 1 {
		t.Fatalf("Gather() families = %d, want 1", len(families))
	}
	metric := families[0].GetMetric()[0].GetHistogram()
	if metric.GetSampleCount() != 2 {
		t.Fatalf("histogram sample count = %d, want 2", metric.GetSampleCount())
	}
	if metric.GetSampleSum() != 30 {
		t.Fatalf("histogram sample sum = %v, want 30", metric.GetSampleSum())
	}
}

func TestNewCounterVecIncompatibleDuplicateDoesNotPanic(t *testing.T) {
	reg := swapDefaultRegistry(t)

	existing := prom.NewCounterVec(prom.CounterOpts{
		Namespace: "test_metric",
		Subsystem: "counter",
		Name:      "conflict_total",
		Help:      "conflicting counter",
	}, []string{"left"})
	reg.MustRegister(existing)
	existing.WithLabelValues("ok").Inc()

	counter := NewCounterVec(&CounterVecOpts{
		Namespace: "test_metric",
		Subsystem: "counter",
		Name:      "conflict_total",
		Help:      "different help and labels",
		Labels:    []string{"right"},
	})

	counter.Inc("ok")

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	if len(families) != 1 {
		t.Fatalf("Gather() families = %d, want 1", len(families))
	}
}

func swapDefaultRegistry(t *testing.T) *prom.Registry {
	t.Helper()

	previousRegisterer := prom.DefaultRegisterer
	previousGatherer := prom.DefaultGatherer
	reg := prom.NewRegistry()
	prom.DefaultRegisterer = reg
	prom.DefaultGatherer = reg

	t.Cleanup(func() {
		prom.DefaultRegisterer = previousRegisterer
		prom.DefaultGatherer = previousGatherer
	})

	return reg
}
