package httperror

import (
	"net/http"
	"testing"

	"goshop/gmicro/errcode"
	apperrors "goshop/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestResponseForGRPCDomainErrorsNeverUseUnknownCode(t *testing.T) {
	errcode.RegisterAll()

	tests := []struct {
		name       string
		grpcCode   codes.Code
		message    string
		wantStatus int
	}{
		{name: "user validation", grpcCode: codes.InvalidArgument, message: "password must contain a number", wantStatus: http.StatusBadRequest},
		{name: "goods conflict", grpcCode: codes.Aborted, message: "concurrent update", wantStatus: http.StatusConflict},
		{name: "inventory missing", grpcCode: codes.NotFound, message: "stock not found", wantStatus: http.StatusNotFound},
		{name: "order dependency unavailable", grpcCode: codes.Unavailable, message: "inventory unavailable", wantStatus: http.StatusServiceUnavailable},
		{name: "review validation", grpcCode: codes.FailedPrecondition, message: "order is not reviewable", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := ResponseFor(status.Error(tt.grpcCode, tt.message))
			if response.Status != tt.wantStatus {
				t.Errorf("ResponseFor(%s) status = %d, want %d", tt.grpcCode, response.Status, tt.wantStatus)
			}
			if response.Code == 1 {
				t.Errorf("ResponseFor(%s) code = %d, want a public non-unknown code", tt.grpcCode, response.Code)
			}
			if response.Message == tt.message {
				t.Errorf("ResponseFor(%s) exposed untrusted gRPC message %q", tt.grpcCode, tt.message)
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
