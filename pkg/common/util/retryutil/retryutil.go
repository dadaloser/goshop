package retryutil

import (
	"context"
	"errors"
	"time"

	"goshop/pkg/common/contextutil"
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

	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	runCtx := contextutil.Root()
	cancel := func() {}
	if ctx != nil && *ctx != nil {
		runCtx = *ctx
	}
	runCtx, cancel = context.WithTimeout(runCtx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-runCtx.Done():
			if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
				return ErrTimeout
			}
			return runCtx.Err()
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
