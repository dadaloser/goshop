package contextutil

import (
	"context"
	"testing"
	"time"
)

func TestNewOperationHasDeadline(t *testing.T) {
	ctx, cancel := NewOperation(25 * time.Millisecond)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Errorf("NewOperation(25ms) deadline present = %t, want true", ok)
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > time.Second {
		t.Errorf("NewOperation(25ms) deadline remaining = %v, want (0s, 1s]", remaining)
	}
}

func TestNewProcessCancels(t *testing.T) {
	ctx, cancel := NewProcess()
	cancel()

	if err := ctx.Err(); err != context.Canceled {
		t.Errorf("NewProcess() context error = %v, want %v", err, context.Canceled)
	}
}

func TestOrProcessPreservesCallerContext(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	ctx, release := OrProcess(parent)
	release()
	if ctx != parent {
		t.Errorf("OrProcess(parent) context = %p, want caller context %p", ctx, parent)
	}
	if err := ctx.Err(); err != nil {
		t.Errorf("OrProcess(parent) context error = %v, want nil", err)
	}
}
