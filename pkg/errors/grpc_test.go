package errors

import (
	"context"
	"net/http"
	"testing"

	grpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGrpcErrorMapsCustomCodeThroughHTTPStatus(t *testing.T) {
	const userNotFoundCode = 990404
	Register(defaultCoder{
		C:    userNotFoundCode,
		HTTP: http.StatusNotFound,
		Ext:  "User not found",
	})

	err := ToGrpcError(WithCode(userNotFoundCode, "select user: record not found"))

	if got := status.Code(err); got != grpcCodes.NotFound {
		t.Fatalf("ToGrpcError() code = %v, want %v", got, grpcCodes.NotFound)
	}
	if got := status.Convert(err).Message(); got != "User not found" {
		t.Fatalf("ToGrpcError() message = %q, want %q", got, "User not found")
	}
}

func TestToGrpcErrorMapsDatabaseErrorsToInternal(t *testing.T) {
	const databaseCode = 990500
	Register(defaultCoder{
		C:    databaseCode,
		HTTP: http.StatusInternalServerError,
		Ext:  "Database error",
	})

	err := ToGrpcError(WrapC(assertAnError{}, databaseCode, "query user"))

	if got := status.Code(err); got != grpcCodes.Internal {
		t.Fatalf("ToGrpcError() code = %v, want %v", got, grpcCodes.Internal)
	}
	if got := status.Convert(err).Message(); got != "Database error" {
		t.Fatalf("ToGrpcError() message = %q, want %q", got, "Database error")
	}
}

func TestToGrpcErrorMapsContextErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want grpcCodes.Code
	}{
		{name: "canceled", err: context.Canceled, want: grpcCodes.Canceled},
		{name: "deadline", err: context.DeadlineExceeded, want: grpcCodes.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status.Code(ToGrpcError(tt.err)); got != tt.want {
				t.Fatalf("ToGrpcError() code = %v, want %v", got, tt.want)
			}
		})
	}
}

type assertAnError struct{}

func (assertAnError) Error() string { return "assert error" }
