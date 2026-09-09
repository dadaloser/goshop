package v1

import (
	"context"
	"testing"
	"time"
)

func TestAccountDeletionOutboxStateContextSurvivesSweepTimeout(t *testing.T) {
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	sweepCtx, cancelSweep := context.WithCancel(runCtx)
	cancelSweep()

	stateCtx, cancelState := accountDeletionOutboxStateContext(runCtx)
	defer cancelState()
	if err := stateCtx.Err(); err != nil {
		t.Fatalf("state context error = %v, want nil", err)
	}
	if _, ok := stateCtx.Deadline(); !ok {
		t.Fatal("state context has no deadline")
	}
	if err := sweepCtx.Err(); err == nil {
		t.Fatal("sweep context error = nil, want cancellation")
	}
	if deadline, ok := stateCtx.Deadline(); !ok || time.Until(deadline) > accountDeletionOutboxStateWriteTimeout {
		t.Fatal("state context deadline is not bounded")
	}
}
