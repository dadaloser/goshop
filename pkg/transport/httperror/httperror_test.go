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
	apperrors.MustRegister(apperrors.Spec{
		Code:    code,
		Kind:    apperrors.KindNotFound,
		Message: "legacy not found",
	})

	response := ResponseFor(apperrors.NewCode(code, "database query failed"))
	if response.Status != http.StatusNotFound {
		t.Fatalf("ResponseFor() status = %d, want %d", response.Status, http.StatusNotFound)
	}
	if response.Message != "legacy not found" {
		t.Fatalf("ResponseFor() message = %q, want legacy not found", response.Message)
	}
}

func TestResponseForAggregateSpec(t *testing.T) {
	err := apperrors.NewAggregate([]error{
		apperrors.New("other failure"),
		apperrors.NewSpec(apperrors.Spec{Code: 990203, Kind: apperrors.KindNotFound, Message: "safe message"}, "query user"),
	})

	response := ResponseFor(err)
	if response.Status != http.StatusNotFound || response.Code != 990203 || response.Message != "safe message" {
		t.Fatalf("ResponseFor(Aggregate) = %#v, want not-found public response", response)
	}
}
