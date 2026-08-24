package gin

import (
	"net/http/httptest"
	"testing"

	"goshop/pkg/log"

	gingonic "github.com/gin-gonic/gin"
)

func TestContextCopiesLogCorrelationValues(t *testing.T) {
	gingonic.SetMode(gingonic.TestMode)
	c, _ := gingonic.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set(log.KeyRequestID, "req-123")
	c.Set(log.KeyUserID, uint64(42))

	ctx := Context(c)
	if got := ctx.Value(log.KeyRequestID); got != "req-123" {
		t.Errorf("Context() request ID = %v, want %q", got, "req-123")
	}
	if got := ctx.Value(log.KeyUserID); got != uint64(42) {
		t.Errorf("Context() user ID = %v, want %d", got, uint64(42))
	}
}

func TestContextNilGinContextReturnsBackground(t *testing.T) {
	if got := Context(nil); got == nil {
		t.Error("Context(nil) = nil, want non-nil context")
	}
}
