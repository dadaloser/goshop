package serverinterceptors

import (
	"context"
	"goshop/gmicro/core/metric"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"
)

const serverNamespace = "rpc_server"

/*
两个基本指标。 1. 每个请求的耗时(histogram) 2. 每个请求的状态计数器(counter)
/user 状态码 有label 主要是状态码
*/

var (
	metricServerReqDur = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: serverNamespace,
		Subsystem: "requests",
		Name:      "goshop_duration_ms",
		Help:      "rpc server requests duration(ms).",
		Labels:    []string{"method"},
		Buckets: []float64{
			5, 10, 25, 50, 100, 250, 500, 1000,
			2000, 5000, 10000, 15000, 30000, 60000,
			120000, 300000,
		}, //强化延迟策略
	})

	metricServerReqCodeTotal = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: serverNamespace,
		Subsystem: "requests",
		Name:      "goshop_code_total",
		Help:      "rpc server requests code count.",
		Labels:    []string{"method", "code"},
	})

	metricServerReqInflight = metric.NewGaugeVec(&metric.GaugeVecOpts{
		Namespace: serverNamespace,
		Subsystem: "requests",
		Name:      "goshop_inflight",
		Help:      "rpc server inflight requests.",
		Labels:    []string{"method"},
	})
)

func UnaryPrometheusInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {

	startTime := time.Now()
	metricServerReqInflight.Inc(info.FullMethod)
	defer metricServerReqInflight.Add(-1, info.FullMethod)
	defer func() {
		if recovered := recover(); recovered != nil {
			observeServerRequest(info.FullMethod, startTime, codes.Internal)
			panic(recovered)
		}
		observeServerRequest(info.FullMethod, startTime, status.Code(err))
	}()
	resp, err = handler(ctx, req)
	return resp, err
}

// StreamPrometheusInterceptor records one observation for the complete stream
// lifetime. Method names are bounded by the registered RPC surface.
func StreamPrometheusInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) (err error) {
	startTime := time.Now()
	metricServerReqInflight.Inc(info.FullMethod)
	defer metricServerReqInflight.Add(-1, info.FullMethod)
	defer func() {
		if recovered := recover(); recovered != nil {
			observeServerRequest(info.FullMethod, startTime, codes.Internal)
			panic(recovered)
		}
		observeServerRequest(info.FullMethod, startTime, status.Code(err))
	}()

	err = handler(srv, stream)
	return err
}

func observeServerRequest(method string, started time.Time, code codes.Code) {
	metricServerReqDur.Observe(int64(time.Since(started)/time.Millisecond), method)
	metricServerReqCodeTotal.Inc(method, strconv.Itoa(int(code)))
}
