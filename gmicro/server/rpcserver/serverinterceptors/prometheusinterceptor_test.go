package serverinterceptors

import (
	"context"
	"strconv"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryPrometheusInterceptorObservesMappedError(t *testing.T) {
	const (
		method = "/test.MetricsService/MappedNotFound"
	)

	_, _ = UnaryPrometheusInterceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: method},
		func(ctx context.Context, req any) (any, error) {
			return UnaryErrorInterceptorWithMapper(func(error) error {
				return status.Error(codes.NotFound, "missing")
			})(ctx, req, &grpc.UnaryServerInfo{FullMethod: method}, func(context.Context, any) (any, error) {
				return nil, status.Error(codes.Internal, "storage failure")
			})
		},
	)

	if got := gatheredCounterValue(t, "rpc_server_requests_code_total", map[string]string{
		"method": method,
		"code":   strconv.Itoa(int(codes.NotFound)),
	}); got < 1 {
		t.Fatalf("gRPC mapped status counter = %v, want at least 1", got)
	}
}

func TestPanicMetricsRecordInternalForUnaryAndStream(t *testing.T) {
	const unaryMethod = "/test.MetricsService/UnaryPanic"
	_, unaryErr := UnaryCrashInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: unaryMethod},
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return UnaryPrometheusInterceptor(ctx, req, &grpc.UnaryServerInfo{FullMethod: unaryMethod},
				func(context.Context, interface{}) (interface{}, error) { panic("unary boom") })
		})
	if got := status.Code(unaryErr); got != codes.Internal {
		t.Fatalf("unary panic code = %v, want Internal", got)
	}
	if got := gatheredCounterValue(t, "rpc_server_requests_code_total", map[string]string{"method": unaryMethod, "code": strconv.Itoa(int(codes.Internal))}); got != 1 {
		t.Fatalf("unary panic metric = %v, want 1", got)
	}

	const streamMethod = "/test.MetricsService/StreamPanic"
	streamErr := StreamCrashInterceptor(nil, testServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: streamMethod},
		func(srv interface{}, stream grpc.ServerStream) error {
			return StreamPrometheusInterceptor(srv, stream, &grpc.StreamServerInfo{FullMethod: streamMethod},
				func(interface{}, grpc.ServerStream) error { panic("stream boom") })
		})
	if got := status.Code(streamErr); got != codes.Internal {
		t.Fatalf("stream panic code = %v, want Internal", got)
	}
	if got := gatheredCounterValue(t, "rpc_server_requests_code_total", map[string]string{"method": streamMethod, "code": strconv.Itoa(int(codes.Internal))}); got != 1 {
		t.Fatalf("stream panic metric = %v, want 1", got)
	}
	if got := gatheredHistogramCount(t, "rpc_server_requests_duration_ms", unaryMethod); got != 1 {
		t.Fatalf("unary panic latency count = %v, want 1", got)
	}
	if got := gatheredHistogramCount(t, "rpc_server_requests_duration_ms", streamMethod); got != 1 {
		t.Fatalf("stream panic latency count = %v, want 1", got)
	}
}

func gatheredHistogramCount(t *testing.T, familyName, method string) uint64 {
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
			for _, label := range metric.GetLabel() {
				if label.GetName() == "method" && label.GetValue() == method {
					return metric.GetHistogram().GetSampleCount()
				}
			}
		}
	}
	return 0
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
