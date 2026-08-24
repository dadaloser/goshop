package grpcerror

import (
	"context"
	"testing"

	apperrors "goshop/pkg/errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapPreservesBusinessWireContract(t *testing.T) {
	spec := apperrors.Spec{Code: 990404, Kind: apperrors.KindNotFound, Message: "user not found"}
	apperrors.MustRegister(spec)
	err := Map(apperrors.NewSpec(spec, "select user: record not found"))
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("Map() code = %v, want %v", got, codes.NotFound)
	}
	info, ok := status.Convert(err).Details()[0].(*errdetails.ErrorInfo)
	if !ok || info.GetReason() != "GOSHOP_BUSINESS_ERROR" || info.GetMetadata()["business_code"] != "990404" {
		t.Fatalf("Map() details = %#v, want business error detail", status.Convert(err).Details())
	}
}

func TestMapPreservesTransportAndContextErrors(t *testing.T) {
	if got := status.Code(Map(status.Error(codes.NotFound, "missing"))); got != codes.NotFound {
		t.Fatalf("Map(status) code = %v, want NotFound", got)
	}
	if got := status.Code(Map(context.Canceled)); got != codes.Canceled {
		t.Fatalf("Map(context.Canceled) code = %v, want Canceled", got)
	}
}
