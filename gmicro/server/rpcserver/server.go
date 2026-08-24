package rpcserver

import (
	"context"
	"sync"

	"goshop/gmicro/server/rpcserver/resolver/discovery"
	srvintc "goshop/gmicro/server/rpcserver/serverinterceptors"
	"goshop/pkg/host"
	"goshop/pkg/log"
	"net"
	"net/url"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	apimd "goshop/api/metadata"
)

type ServerOption func(o *Server)

type Server struct {
	*grpc.Server

	address                         string
	unaryInts                       []grpc.UnaryServerInterceptor
	streamInts                      []grpc.StreamServerInterceptor
	grpcOpts                        []grpc.ServerOption
	lis                             net.Listener
	unaryTimeout                    time.Duration
	streamMaxLifetime               time.Duration
	maxConcurrentApplicationStreams int
	maxConcurrentUnaryRequests      int

	health         *health.Server
	metadata       *apimd.Server
	endpoint       *url.URL
	ready          chan struct{}
	readyOnce      sync.Once
	readinessCheck func() error

	tlsEnabled         bool
	enableMetrics      bool
	enableReflection   bool
	productionDefaults bool

	securityPolicy *SecurityPolicy
	errorMapper    srvintc.ErrorMapper
}

func (s *Server) Endpoint() *url.URL {
	return s.endpoint
}

func (s *Server) Address() string {
	return s.address
}

func (s *Server) Ready() <-chan struct{} {
	return s.ready
}

func NewServer(opts ...ServerOption) *Server {
	srv, err := NewServerE(opts...)
	if err != nil {
		panic(err)
	}
	return srv
}

func NewServerE(opts ...ServerOption) (*Server, error) {
	srv := &Server{
		address:                         ":0",
		health:                          health.NewServer(),
		ready:                           make(chan struct{}),
		enableMetrics:                   true,
		productionDefaults:              true,
		maxConcurrentApplicationStreams: 256,
		maxConcurrentUnaryRequests:      256,
		//timeout: 1 * time.Second,
	}

	for _, o := range opts {
		o(srv)
	}

	if srv.securityPolicy != nil {
		if srv.tlsEnabled {
			return nil, errServerSecurityAlreadyConfigured
		}
		tlsConfig, loadErr := srv.securityPolicy.LoadServerTLSConfig()
		if loadErr != nil {
			return nil, loadErr
		}
		applyServerTLSConfig(srv, tlsConfig)
	}

	// Keep crash recovery outermost. Metrics must wrap error conversion so it
	// observes the final gRPC status instead of classifying project errors as
	// codes.Unknown.
	unaryInts := []grpc.UnaryServerInterceptor{
		srvintc.UnaryCrashInterceptor,
	}
	streamInts := []grpc.StreamServerInterceptor{srvintc.StreamCrashInterceptor}

	if srv.enableMetrics {
		unaryInts = append(unaryInts, srvintc.UnaryPrometheusInterceptor)
		streamInts = append(streamInts, srvintc.StreamPrometheusInterceptor)
	}
	unaryInts = append(unaryInts, srvintc.UnaryErrorInterceptorWithMapper(srv.errorMapper))
	streamInts = append(streamInts, srvintc.StreamErrorInterceptorWithMapper(srv.errorMapper))
	if srv.securityPolicy != nil {
		unaryInts = append(unaryInts, srvintc.UnaryClientIdentityInterceptor(srv.securityPolicy.AllowedClientIdentities))
		streamInts = append(streamInts, srvintc.StreamClientIdentityInterceptor(srv.securityPolicy.AllowedClientIdentities))
	}
	if srv.maxConcurrentUnaryRequests > 0 {
		unaryInts = append(unaryInts, srvintc.UnaryConcurrencyInterceptor(srv.maxConcurrentUnaryRequests))
	}

	if srv.unaryTimeout > 0 {
		unaryInts = append(unaryInts, srvintc.UnaryTimeoutInterceptor(srv.unaryTimeout))
	}
	if srv.streamMaxLifetime > 0 {
		streamInts = append(streamInts, srvintc.StreamTimeoutInterceptor(srv.streamMaxLifetime))
	}
	if srv.maxConcurrentApplicationStreams > 0 {
		streamInts = append(streamInts, srvintc.StreamConcurrencyInterceptor(srv.maxConcurrentApplicationStreams))
	}

	if len(srv.unaryInts) > 0 {
		unaryInts = append(unaryInts, srv.unaryInts...)
	}
	if len(srv.streamInts) > 0 {
		streamInts = append(streamInts, srv.streamInts...)
	}

	//把我们传入的拦截器转换成grpc的ServerOption
	grpcOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unaryInts...),
		grpc.ChainStreamInterceptor(streamInts...),
		//注意:链路追踪拦截器需要独立出来
		grpc.StatsHandler(otelgrpc.NewServerHandler())}

	if srv.productionDefaults {
		grpcOpts = append(grpcOpts, productionServerOptions()...)
	}

	//把用户自己传入的grpc.ServerOption放在一起
	if len(srv.grpcOpts) > 0 {
		grpcOpts = append(grpcOpts, srv.grpcOpts...)
	}
	srv.grpcOpts = grpcOpts

	srv.Server = grpc.NewServer(grpcOpts...)

	//注册metadata的Server
	srv.metadata = apimd.NewServer(srv.Server)

	//解析address
	err := srv.listenAndEndpoint()
	if err != nil {
		return nil, err
	}

	//注册health
	grpc_health_v1.RegisterHealthServer(srv.Server, srv.health)
	apimd.RegisterMetadataServer(srv.Server, srv.metadata)
	if srv.enableReflection {
		reflection.Register(srv.Server)
	}
	//可以支持用户直接通过grpc的一个接口查看当前支持的所有的rpc服务

	return srv, nil
}

func WithAddress(address string) ServerOption {
	return func(s *Server) {
		s.address = address
	}
}

// WithErrorMapper configures the application's gRPC error protocol. When it
// is unset, rpcserver emits only standard gRPC status errors.
func WithErrorMapper(mapper srvintc.ErrorMapper) ServerOption {
	return func(s *Server) {
		s.errorMapper = mapper
	}
}

// WithReadinessCheck controls the gRPC health status for an external dependency.
func WithReadinessCheck(check func() error) ServerOption {
	return func(s *Server) {
		s.readinessCheck = check
	}
}

func WithMetrics(metric bool) ServerOption {
	return func(s *Server) {
		s.enableMetrics = metric
	}
}

func WithReflection(enable bool) ServerOption {
	return func(s *Server) {
		s.enableReflection = enable
	}
}

// WithTimeout is retained for source compatibility and configures unary RPCs.
// New code should use WithUnaryTimeout.
func WithTimeout(timeout time.Duration) ServerOption {
	return WithUnaryTimeout(timeout)
}

func WithUnaryTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.unaryTimeout = timeout
	}
}

func WithStreamMaxLifetime(timeout time.Duration) ServerOption {
	return func(s *Server) { s.streamMaxLifetime = timeout }
}

// WithApplicationStreamConcurrency applies a fail-fast bulkhead before stream
// handlers start. This complements gRPC's transport-level concurrent stream
// limit with an explicit application capacity limit.
func WithApplicationStreamConcurrency(maxConcurrent int) ServerOption {
	return func(s *Server) {
		if maxConcurrent > 0 {
			s.maxConcurrentApplicationStreams = maxConcurrent
		}
	}
}

// WithApplicationUnaryConcurrency applies a process-wide fail-fast bulkhead
// before unary handlers start.
func WithApplicationUnaryConcurrency(maxConcurrent int) ServerOption {
	return func(s *Server) {
		if maxConcurrent > 0 {
			s.maxConcurrentUnaryRequests = maxConcurrent
		}
	}
}

func WithLis(lis net.Listener) ServerOption {
	return func(s *Server) {
		s.lis = lis
	}
}

func WithUnaryInterceptor(in ...grpc.UnaryServerInterceptor) ServerOption {
	return func(s *Server) {
		s.unaryInts = in
	}
}

func WithStreamInterceptor(in ...grpc.StreamServerInterceptor) ServerOption {
	return func(s *Server) {
		s.streamInts = in
	}
}

func WithOptions(opts ...grpc.ServerOption) ServerOption {
	return func(s *Server) {
		s.grpcOpts = opts
	}
}

func WithMaxConcurrentStreams(max uint32) ServerOption {
	return func(s *Server) {
		if max > 0 {
			s.grpcOpts = append(s.grpcOpts, grpc.MaxConcurrentStreams(max))
		}
	}
}

func WithKeepaliveParams(params keepalive.ServerParameters) ServerOption {
	return func(s *Server) {
		s.grpcOpts = append(s.grpcOpts, grpc.KeepaliveParams(params))
	}
}

func WithKeepaliveEnforcementPolicy(policy keepalive.EnforcementPolicy) ServerOption {
	return func(s *Server) {
		s.grpcOpts = append(s.grpcOpts, grpc.KeepaliveEnforcementPolicy(policy))
	}
}

func WithProductionDefaults() ServerOption {
	return func(s *Server) {
		s.productionDefaults = true
	}
}

func productionServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.MaxConcurrentStreams(1024),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 10 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}
}

// 完成ip和端口的提取
func (s *Server) listenAndEndpoint() error {
	if s.lis == nil {
		lis, err := net.Listen("tcp", s.address)
		if err != nil {
			return err
		}
		s.lis = lis
	}
	addr, err := host.Extract(s.address, s.lis)
	if err != nil {
		_ = s.lis.Close()
		return err
	}
	s.endpoint = discovery.NewEndpoint("grpc", addr, s.tlsEnabled)
	return nil
}

// Start 启动grpc的服务
func (s *Server) Start(ctx context.Context) error {
	log.Infof("[grpc] server listening on: %s", s.lis.Addr().String())
	s.health.Resume()
	s.updateReadiness()
	if s.readinessCheck != nil {
		go s.monitorReadiness(ctx)
	}
	s.readyOnce.Do(func() {
		close(s.ready)
	})
	return s.Serve(s.lis)
}

func (s *Server) monitorReadiness(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateReadiness()
		}
	}
}

func (s *Server) updateReadiness() {
	if s.readinessCheck == nil || s.readinessCheck() == nil {
		s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
		return
	}
	s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
}

func (s *Server) Stop(ctx context.Context) error {
	//设置服务的状态为not_serving，防止接收新的请求过来
	s.health.Shutdown()
	//GracefulStop() 现在会受 ctx 控制，超时后强制 Stop()，避免退出卡死
	done := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		s.Server.Stop()
	}
	log.Infof("[grpc] server stopped")
	return nil
}
