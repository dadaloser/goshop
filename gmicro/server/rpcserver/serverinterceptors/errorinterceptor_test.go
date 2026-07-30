package serverinterceptors

import (
	"context"
	"testing"

	apperrors "goshop/pkg/errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryErrorInterceptorConvertsProjectError(t *testing.T) {
	resp, err := UnaryErrorInterceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Error"},
		func(context.Context, interface{}) (interface{}, error) {
			return nil, apperrors.NewCode(1, "database exploded")
		},
	)

	if resp != nil {
		t.Fatalf("UnaryErrorInterceptor() resp = %v, want nil", resp)
	}
	if got := status.Code(err); got != codes.Internal {
		t.Fatalf("UnaryErrorInterceptor() code = %v, want %v", got, codes.Internal)
	}
}

func TestUnaryErrorInterceptorPreservesStatusError(t *testing.T) {
	_, err := UnaryErrorInterceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/NotFound"},
		func(context.Context, interface{}) (interface{}, error) {
			return nil, status.Error(codes.NotFound, "missing")
		},
	)

	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("UnaryErrorInterceptor() code = %v, want %v", got, codes.NotFound)
	}
}

func TestToGRPCErrorMapsSpecKinds(t *testing.T) {
	tests := []struct {
		name string
		kind apperrors.Kind
		want codes.Code
	}{
		{name: "not found", kind: apperrors.KindNotFound, want: codes.NotFound},
		{name: "conflict", kind: apperrors.KindConflict, want: codes.Aborted},
		{name: "unavailable", kind: apperrors.KindUnavailable, want: codes.Unavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := apperrors.NewSpec(apperrors.Spec{
				Code:    990101,
				Kind:    tt.kind,
				Message: "safe message",
			}, "database query failed")

			got := toGRPCError(err)
			if status.Code(got) != tt.want {
				t.Fatalf("toGRPCError() code = %v, want %v", status.Code(got), tt.want)
			}
			if status.Convert(got).Message() != "safe message" {
				t.Fatalf("toGRPCError() message = %q, want safe message", status.Convert(got).Message())
			}
		})
	}
}

func TestToGRPCErrorKeepsLegacyCodeMapping(t *testing.T) {
	const userNotFoundCode = 990404
	apperrors.MustRegister(apperrors.Spec{
		Code:    userNotFoundCode,
		Kind:    apperrors.KindNotFound,
		Message: "User not found",
	})

	err := toGRPCError(apperrors.NewCode(userNotFoundCode, "select user: record not found"))
	if status.Code(err) != codes.NotFound {
		t.Fatalf("toGRPCError() code = %v, want %v", status.Code(err), codes.NotFound)
	}
	if status.Convert(err).Message() != "User not found" {
		t.Fatalf("toGRPCError() message = %q, want User not found", status.Convert(err).Message())
	}
	if len(status.Convert(err).Details()) != 1 {
		t.Fatalf("toGRPCError() details = %#v, want one business detail", status.Convert(err).Details())
	}
	info, ok := status.Convert(err).Details()[0].(*errdetails.ErrorInfo)
	if !ok || info.GetReason() != "GOSHOP_BUSINESS_ERROR" || info.GetMetadata()["business_code"] != "990404" {
		t.Fatalf("toGRPCError() ErrorInfo = %#v, want GOSHOP_BUSINESS_ERROR code 990404", status.Convert(err).Details()[0])
	}
}

func TestToGRPCErrorMapsAggregateSpec(t *testing.T) {
	err := apperrors.NewAggregate([]error{
		apperrors.New("other failure"),
		apperrors.NewSpec(apperrors.Spec{Code: 990405, Kind: apperrors.KindNotFound, Message: "safe message"}, "query user"),
	})

	got := toGRPCError(err)
	if status.Code(got) != codes.NotFound || status.Convert(got).Message() != "safe message" {
		t.Fatalf("toGRPCError(Aggregate) = %v, want not-found safe message", got)
	}
}

func TestToGRPCErrorMapsContextErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "canceled", err: context.Canceled, want: codes.Canceled},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: codes.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status.Code(toGRPCError(tt.err)); got != tt.want {
				t.Fatalf("toGRPCError() code = %v, want %v", got, tt.want)
			}
		})
	}
}
