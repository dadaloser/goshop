package httperror

import (
	"net/http"
	"testing"

	apperrors "goshop/pkg/errors"
)

func TestResponseForSpec(t *testing.T) {
	tests := []struct {
		name string
		kind apperrors.Kind
		want int
	}{
		{name: "invalid argument", kind: apperrors.KindInvalidArgument, want: http.StatusBadRequest},
		{name: "not found", kind: apperrors.KindNotFound, want: http.StatusNotFound},
		{name: "conflict", kind: apperrors.KindConflict, want: http.StatusConflict},
		{name: "rate limited", kind: apperrors.KindRateLimited, want: http.StatusTooManyRequests},
		{name: "unavailable", kind: apperrors.KindUnavailable, want: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := ResponseFor(apperrors.NewSpec(apperrors.Spec{
				Code:      990201,
				Kind:      tt.kind,
				Message:   "safe message",
				Reference: "https://example.test/errors",
			}, "database query failed"))

			if response.Status != tt.want {
				t.Fatalf("ResponseFor() status = %d, want %d", response.Status, tt.want)
			}
			if response.Code != 990201 {
				t.Fatalf("ResponseFor() code = %d, want 990201", response.Code)
			}
			if response.Message != "safe message" {
				t.Fatalf("ResponseFor() message = %q, want safe message", response.Message)
			}
		})
	}
}

func TestResponseForLegacyCode(t *testing.T) {
	const code = 990202
	apperrors.Register(testCoder{
		code:    code,
		kind:    apperrors.KindNotFound,
		message: "legacy not found",
	})

	response := ResponseFor(apperrors.NewCode(code, "database query failed"))
	if response.Status != http.StatusNotFound {
		t.Fatalf("ResponseFor() status = %d, want %d", response.Status, http.StatusNotFound)
	}
	if response.Message != "legacy not found" {
		t.Fatalf("ResponseFor() message = %q, want legacy not found", response.Message)
	}
}

type testCoder struct {
	code    int
	kind    apperrors.Kind
	message string
}

func (c testCoder) Code() int            { return c.code }
func (c testCoder) String() string       { return c.message }
func (c testCoder) Reference() string    { return "" }
func (c testCoder) Kind() apperrors.Kind { return c.kind }
