package serverinterceptors

import (
	"context"
	"strconv"
	"testing"

	apperrors "goshop/pkg/errors"

	prom "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestUnaryPrometheusInterceptorObservesConvertedProjectError(t *testing.T) {
	const (
		method       = "/test.MetricsService/ConvertedNotFound"
		businessCode = 991404
	)
	spec := apperrors.Spec{Code: businessCode, Kind: apperrors.KindNotFound, Message: "not found"}
	apperrors.MustRegister(spec)

	_, _ = UnaryPrometheusInterceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: method},
		func(ctx context.Context, req any) (any, error) {
			return UnaryErrorInterceptor(ctx, req, &grpc.UnaryServerInfo{FullMethod: method}, func(context.Context, any) (any, error) {
				return nil, apperrors.NewSpec(spec, "record missing")
			})
		},
	)

	if got := gatheredCounterValue(t, "rpc_server_requests_goshop_code_total", map[string]string{
		"method": method,
		"code":   strconv.Itoa(int(codes.NotFound)),
	}); got < 1 {
		t.Fatalf("gRPC converted status counter = %v, want at least 1", got)
	}
	if got := gatheredCounterValue(t, "rpc_server_requests_goshop_code_total", map[string]string{
		"method": method,
		"code":   strconv.Itoa(int(codes.Unknown)),
	}); got != 0 {
		t.Fatalf("gRPC unknown status counter = %v, want 0", got)
	}
}

func gatheredCounterValue(t *testing.T, familyName string, labels map[string]string) float64 {
	t.Helper()
	families, err := prom.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("prometheus Gather() error = %v, want nil", err)
	}
	for _, family := range families {
		if family.GetName() != familyName {
			continue
		}
		for _, metric := range family.GetMetric() {
			matched := true
			for name, want := range labels {
				found := false
				for _, pair := range metric.GetLabel() {
					if pair.GetName() == name && pair.GetValue() == want {
						found = true
						break
					}
				}
				if !found {
					matched = false
					break
				}
			}
			if matched {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}
