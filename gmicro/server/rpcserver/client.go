package rpcserver

import (
	"context"
	"fmt"
	"time"

	"goshop/gmicro/registry"
	"goshop/gmicro/resilience"
	"goshop/gmicro/server/rpcserver/clientinterceptors"
	"goshop/gmicro/server/rpcserver/resolver/discovery"
	"goshop/pkg/log"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	grpcinsecure "google.golang.org/grpc/credentials/insecure"
)

const selectorBalancerName = "selector"

type ClientOption func(o *clientOptions)

type clientOptions struct {
	endpoint string
	timeout  time.Duration
	// discovery接口
	discovery      registry.Discovery
	unaryInts      []grpc.UnaryClientInterceptor
	streamInts     []grpc.StreamClientInterceptor
	rpcOpts        []grpc.DialOption
	balancerName   string
	log            log.LogHelper
	enableTracing  bool
	enableMetrics  bool
	connectProbe   bool
	connectTimeout time.Duration
	resilience     *resilience.Options

	transportCredentials credentials.TransportCredentials
	securityPolicy       *SecurityPolicy
}

func WithEnableTracing(enable bool) ClientOption {
	return func(o *clientOptions) {
		o.enableTracing = enable
	}
}

// WithConnectProbe enables startup connectivity probing. When enabled, Dial
// returns only after the connection reaches READY or the probe timeout expires.
func WithConnectProbe(enable bool) ClientOption {
	return func(o *clientOptions) {
		o.connectProbe = enable
	}
}

// WithConnectTimeout sets the startup connectivity probe timeout.
func WithConnectTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.connectTimeout = timeout
	}
}

// 设置地址
func WithEndpoint(endpoint string) ClientOption {
	return func(o *clientOptions) {
		o.endpoint = endpoint
	}
}

// 设置超时时间
func WithClientTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

// WithClientResilience configures outbound RPC timeout, isolation, and circuit breaking.
func WithClientResilience(options *resilience.Options) ClientOption {
	return func(o *clientOptions) {
		o.resilience = options
	}
}

// 设置服务发现
func WithDiscovery(d registry.Discovery) ClientOption {
	return func(o *clientOptions) {
		o.discovery = d
	}
}

// 设置拦截器
func WithClientUnaryInterceptor(in ...grpc.UnaryClientInterceptor) ClientOption {
	return func(o *clientOptions) {
		o.unaryInts = in
	}
}

// 设置stream拦截器
func WithClientStreamInterceptor(in ...grpc.StreamClientInterceptor) ClientOption {
	return func(o *clientOptions) {
		o.streamInts = in
	}
}

// 设置grpc的dial选项
func WithClientOptions(opts ...grpc.DialOption) ClientOption {
	return func(o *clientOptions) {
		o.rpcOpts = opts
	}
}

// 设置负载均衡器
func WithBalancerName(name string) ClientOption {
	return func(o *clientOptions) {
		o.balancerName = name
	}
}

func DialInsecure(ctx context.Context, opts ...ClientOption) (*grpc.ClientConn, error) {
	return dial(ctx, true, opts...)
}

func Dial(ctx context.Context, opts ...ClientOption) (*grpc.ClientConn, error) {
	return dial(ctx, false, opts...)
}

// DialDiscoveryInsecure dials a registry-discovered service with production defaults:
// startup probing enabled and the framework selector balancer registered.
func DialDiscoveryInsecure(ctx context.Context, opts ...ClientOption) (*grpc.ClientConn, error) {
	return dialDiscovery(ctx, true, opts...)
}

// DialDiscovery dials a registry-discovered service with production defaults.
func DialDiscovery(ctx context.Context, opts ...ClientOption) (*grpc.ClientConn, error) {
	return dialDiscovery(ctx, false, opts...)
}

func dialDiscovery(ctx context.Context, insecure bool, opts ...ClientOption) (*grpc.ClientConn, error) {
	options := clientOptions{
		connectProbe: true,
		balancerName: selectorBalancerName,
	}
	for _, opt := range opts {
		opt(&options)
	}
	if options.discovery == nil {
		return nil, fmt.Errorf("rpc discovery is required")
	}
	InitBuilder()

	discoveryOpts := []ClientOption{
		WithDiscovery(options.discovery),
		WithBalancerName(options.balancerName),
		WithConnectProbe(options.connectProbe),
	}
	opts = append(discoveryOpts, opts...)
	return dial(ctx, insecure, opts...)
}

func dial(ctx context.Context, insecure bool, opts ...ClientOption) (*grpc.ClientConn, error) {
	options := clientOptions{
		timeout:        2000 * time.Millisecond,
		connectTimeout: 5 * time.Second,
		balancerName:   "round_robin",
		enableTracing:  true,
	}

	for _, o := range opts {
		o(&options)
	}

	if options.securityPolicy != nil {
		if options.transportCredentials != nil {
			return nil, errClientSecurityAlreadyConfigured
		}
		tlsConfig, err := options.securityPolicy.LoadClientTLSConfig()
		if err != nil {
			return nil, err
		}
		applyClientTLSConfig(&options, tlsConfig)
	}

	resilienceOptions := options.resilience
	if resilienceOptions == nil {
		resilienceOptions = resilience.NewOptions()
		resilienceOptions.Timeout = options.timeout
	}
	sentinelInterceptor, err := clientinterceptors.SentinelInterceptor(resilienceOptions)
	if err != nil {
		return nil, fmt.Errorf("create rpc resilience interceptor: %w", err)
	}
	ints := []grpc.UnaryClientInterceptor{sentinelInterceptor}

	if options.enableMetrics {
		ints = append(ints, clientinterceptors.PrometheusInterceptor())
	}

	var streamInts []grpc.StreamClientInterceptor

	if len(options.unaryInts) > 0 {
		ints = append(ints, options.unaryInts...)
	}
	if len(options.streamInts) > 0 {
		streamInts = append(streamInts, options.streamInts...)
	}

	grpcOpts := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "` + options.balancerName + `"}`),
		grpc.WithChainUnaryInterceptor(ints...),
		grpc.WithChainStreamInterceptor(streamInts...),
	}

	//是否开启链路追踪
	if options.enableTracing {
		grpcOpts = append(grpcOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler())) // OpenTelemetry 的 StatsHandler 在这里，作为独立的 DialOption
	}

	//TODO 服务发现的选项
	if options.discovery != nil {
		grpcOpts = append(grpcOpts, grpc.WithResolvers(
			discovery.NewBuilder(
				options.discovery,
				discovery.WithInsecure(insecure),
			),
		))
	}

	if insecure {
		if options.transportCredentials != nil {
			return nil, fmt.Errorf("rpc TLS credentials cannot be combined with insecure dial")
		}
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(grpcinsecure.NewCredentials()))
	} else {
		if options.transportCredentials == nil {
			return nil, fmt.Errorf("rpc TLS credentials are required for secure dial")
		}
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(options.transportCredentials))
	}

	if len(options.rpcOpts) > 0 {
		grpcOpts = append(grpcOpts, options.rpcOpts...)
	}

	conn, err := grpc.NewClient(options.endpoint, grpcOpts...)
	if err != nil {
		return nil, err
	}
	if options.connectProbe {
		if err := waitForReady(ctx, conn, options.connectTimeout); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return conn, nil
	//return grpc.DialContext(ctx, options.endpoint, grpcOpts...)
}

func waitForReady(ctx context.Context, conn *grpc.ClientConn, timeout time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	probeCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		probeCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if state == connectivity.Shutdown {
			return fmt.Errorf("grpc client connection shutdown before ready")
		}
		if !conn.WaitForStateChange(probeCtx, state) {
			return fmt.Errorf("grpc client connection not ready before startup probe timeout: %w", probeCtx.Err())
		}
	}
}
