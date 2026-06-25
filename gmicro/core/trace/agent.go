package trace

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"

	"goshop/pkg/log"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

/*
初始化不同的export的设置
*/

const (
	kindJaeger = "jaeger"
	kindZipkin = "zipkin"
)

var (
	//set ,struct 空结构体不占内存， zerobase
	agents = make(map[string]struct{})
	lock   sync.Mutex
)

func InitAgent(o Options) {
	//防止反复调用
	lock.Lock()
	defer lock.Unlock()

	_, ok := agents[o.Endpoint]
	if ok {
		return
	}
	err := startAgent(o)
	if err != nil {
		return
	}
	agents[o.Endpoint] = struct{}{}
}

func startAgent(o Options) error {
	var sexp trace.SpanExporter
	var err error

	opts := []trace.TracerProviderOption{
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(o.Sampler))),
		trace.WithResource(resource.NewSchemaless(semconv.ServiceNameKey.String(o.Name))),
	}

	//todo:注意检查zipkin和jaeger的endpoint格式，是否需要协议头，是否需要指定URL路径等
	if len(o.Endpoint) > 0 {
		//sexp, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(o.Endpoint)))
		// 3. 替换 Jaeger 导出器为 OTLP HTTP 导出器
		// 注意：OTLP 默认端口通常是 4318 (HTTP)
		//注意context的传递是否需要tracing信息，是否需要携带traceparent等header
		sexp, err = otlptracehttp.New(context.Background(), otlptracehttp.WithEndpoint(o.Endpoint))
		// 如果是 HTTP，通常需要指定 URL 路径，或者确保 Endpoint 包含协议头
		// otlptracehttp.WithURLPath("/v1/traces"),
		// otlptracehttp.WithInsecure(), // 如果不使用 TLS
		// otlptracehttp.WithHeaders(map[string]string{"Authorization": "Bearer <token>"}
		if err != nil {
			return err
		}
		opts = append(opts, trace.WithBatcher(sexp))
	}

	tp := trace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Errorf("[otel] error: %v", err)
	}))
	return nil
}
