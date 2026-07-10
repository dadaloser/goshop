package metric

import (
	"errors"
	stdlog "log"

	prom "github.com/prometheus/client_golang/prometheus"
)

func registerCounterVec(vec *prom.CounterVec) *prom.CounterVec {
	if vec == nil {
		return nil
	}

	if err := prom.Register(vec); err != nil {
		var alreadyRegistered prom.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prom.CounterVec); ok {
				return existing
			}
		}

		stdlog.Printf("metric: failed to register counter vec: %v", err)
		return nil
	}

	return vec
}

func registerGaugeVec(vec *prom.GaugeVec) *prom.GaugeVec {
	if vec == nil {
		return nil
	}

	if err := prom.Register(vec); err != nil {
		var alreadyRegistered prom.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prom.GaugeVec); ok {
				return existing
			}
		}

		stdlog.Printf("metric: failed to register gauge vec: %v", err)
		return nil
	}

	return vec
}

func registerHistogramVec(vec *prom.HistogramVec) *prom.HistogramVec {
	if vec == nil {
		return nil
	}

	if err := prom.Register(vec); err != nil {
		var alreadyRegistered prom.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prom.HistogramVec); ok {
				return existing
			}
		}

		stdlog.Printf("metric: failed to register histogram vec: %v", err)
		return nil
	}

	return vec
}
