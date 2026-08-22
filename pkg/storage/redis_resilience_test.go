package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"goshop/pkg/resilience"

	"github.com/redis/go-redis/v9"
)

func TestRedisResilienceHookIsolationFallback(t *testing.T) {
	options := resilience.NewOptions()
	options.MaxConcurrency = 1
	options.Timeout = time.Second
	hook, err := newRedisResilienceHook(options)
	if err != nil {
		t.Fatalf("newRedisResilienceHook() error = %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	firstResult := make(chan error, 1)
	process := hook.ProcessHook(func(ctx context.Context, _ redis.Cmder) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	go func() {
		firstResult <- process(t.Context(), redis.NewStringCmd(t.Context(), "get", "key"))
	}()
	<-started

	err = process(t.Context(), redis.NewStringCmd(t.Context(), "get", "key"))
	if !errors.Is(err, resilience.ErrBlocked) {
		t.Fatalf("second call error = %v, want ErrBlocked", err)
	}
	close(release)
	if err := <-firstResult; err != nil {
		t.Fatalf("first call error = %v", err)
	}
}

func TestIsRedisDependencyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "connection error", err: errors.New("connection refused"), want: true},
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "missing key", err: redis.Nil, want: false},
		{name: "caller canceled", err: context.Canceled, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRedisDependencyError(tt.err); got != tt.want {
				t.Fatalf("isRedisDependencyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedisRetryDelayUsesCappedExponentialBackoff(t *testing.T) {
	tests := []struct {
		failures int
		min      time.Duration
		max      time.Duration
	}{
		{failures: 0, min: time.Second, max: time.Second},
		{failures: 1, min: 500 * time.Millisecond, max: time.Second},
		{failures: 2, min: time.Second, max: 2 * time.Second},
		{failures: 6, min: 15 * time.Second, max: 30 * time.Second},
		{failures: 20, min: 15 * time.Second, max: 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run("failures", func(t *testing.T) {
			got := redisRetryDelay(tt.failures)
			if got < tt.min || got > tt.max {
				t.Errorf("redisRetryDelay(%d) = %s, want within [%s, %s]", tt.failures, got, tt.min, tt.max)
			}
		})
	}
}
