package contextutil

import (
	"context"
	"time"
)

const defaultOperationTimeout = 10 * time.Second

// NewOperation returns a timeout-bounded context for a finite operation that
// has no caller context, such as legacy startup code. Call cancel on return.
func NewOperation(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = defaultOperationTimeout
	}
	// This is the sole process-independent root: callers receive a deadline and
	// must release it with the returned cancel function.
	return context.WithTimeout(context.Background(), timeout)
}

// NewProcess returns a cancellable root for a component whose owner explicitly
// invokes the returned cancel function during shutdown (Consul watchers and
// application lifetime). It must never be used for a request or a finite I/O.
func NewProcess() (context.Context, context.CancelFunc) {
	// A process has no request parent. Its lifetime is bounded by its owner via
	// cancel, rather than an arbitrary wall-clock deadline.
	return context.WithCancel(context.Background())
}

// Root returns the documented root only for code that immediately derives a
// bounded child context. NewOperation is preferred whenever possible.
func Root() context.Context {
	return context.Background()
}

// OrProcess returns ctx when supplied, otherwise creates an owned process root.
// The returned cancel is a no-op when ctx already has an owner.
func OrProcess(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx != nil {
		return ctx, func() {}
	}
	return NewProcess()
}
