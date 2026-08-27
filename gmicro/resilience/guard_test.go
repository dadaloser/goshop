package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
	"github.com/alibaba/sentinel-golang/core/isolation"
	"github.com/prometheus/client_golang/prometheus"
)

func TestGuardTimeout(t *testing.T) {
	options := NewOptions()
	options.Enabled = false
	options.Timeout = 10 * time.Millisecond
	guard, err := NewGuard("timeouttest", options, func(error) bool { return true })
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	err = guard.Do(t.Context(), "wait", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Do() error = %v, want context deadline exceeded", err)
	}
}

func TestGuardIsolationFallbackAndRelease(t *testing.T) {
	options := NewOptions()
	options.MaxConcurrency = 1
	guard, err := NewGuard("isolationtest", options, func(error) bool { return true })
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}
	t.Cleanup(func() {
		_ = circuitbreaker.ClearRulesOfResource("isolationtest:query")
		_ = isolation.ClearRulesOfResource("isolationtest:query")
	})

	first, err := guard.Start(t.Context(), "query")
	if err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	if _, err = guard.Start(t.Context(), "query"); !errors.Is(err, ErrBlocked) {
		t.Fatalf("second Start() error = %v, want ErrBlocked", err)
	}
	first.Finish(nil)

	third, err := guard.Start(t.Context(), "query")
	if err != nil {
		t.Fatalf("third Start() after release error = %v", err)
	}
	third.Finish(nil)
}

func TestGuardCircuitRecovery(t *testing.T) {
	labels := map[string]string{
		"dependency": "recoverytest",
		"resource":   "remote",
	}
	recoveriesBefore := metricValue(t, "dependency_resilience_recovery_total", labels)
	options := NewOptions()
	options.MinRequestAmount = 2
	options.ErrorRatio = 0.5
	options.StatInterval = time.Second
	options.RecoveryTimeout = 20 * time.Millisecond
	guard, err := NewGuard("recoverytest", options, func(error) bool { return true })
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}
	t.Cleanup(func() {
		_ = circuitbreaker.ClearRulesOfResource("recoverytest:remote")
		_ = isolation.ClearRulesOfResource("recoverytest:remote")
	})

	dependencyErr := errors.New("dependency failed")
	for range 2 {
		if err := guard.Do(t.Context(), "remote", func(context.Context) error {
			return dependencyErr
		}); !errors.Is(err, dependencyErr) {
			t.Fatalf("Do() error = %v, want dependency error", err)
		}
	}
	if err := guard.Do(t.Context(), "remote", func(context.Context) error { return nil }); !errors.Is(err, ErrBlocked) {
		t.Fatalf("open circuit error = %v, want ErrBlocked", err)
	}

	time.Sleep(30 * time.Millisecond)
	if err := guard.Do(t.Context(), "remote", func(context.Context) error { return nil }); err != nil {
		t.Fatalf("recovery probe error = %v", err)
	}
	if got := metricValue(t, "dependency_resilience_recovery_total", labels); got-recoveriesBefore != 1 {
		t.Fatalf("recovery metric delta = %v, want 1", got-recoveriesBefore)
	}
}

func metricValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, item := range family.Metric {
			matched := true
			for key, expected := range labels {
				found := false
				for _, pair := range item.Label {
					if pair.GetName() == key && pair.GetValue() == expected {
						found = true
						break
					}
				}
				if !found {
					matched = false
					break
				}
			}
			if matched && item.Counter != nil {
				return item.Counter.GetValue()
			}
		}
	}
	return 0
}
