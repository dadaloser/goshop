package retryutil

import (
	"context"
	"errors"
	"time"
)

var ErrRetryable = errors.New("retry")
var ErrTimeout = errors.New("timeout")

func RetryUntilTimeout(ctx *context.Context, interval time.Duration, timeout time.Duration, do func() error) error {
	if do == nil {
		return nil
	}

	err := do()
	if err == nil {
		return nil
	}

	if !errors.Is(err, ErrRetryable) {
		return err
	}

	if interval <= 0 {
		interval = time.Millisecond
	}

	runCtx := context.Background()
	if ctx != nil && *ctx != nil {
		runCtx = *ctx
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var timeoutCh <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		timeoutCh = timer.C
	}

	for {
		select {
		case <-runCtx.Done():
			return runCtx.Err()
		case <-timeoutCh:
			return ErrTimeout
		case <-ticker.C:
			err := do()
			if err == nil {
				return nil
			}
			if !errors.Is(err, ErrRetryable) {
				return err
			}
		}
	}
}
