package rpcserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"goshop/gmicro/server/rpcserver/resolver/discovery"
	srvintc "goshop/gmicro/server/rpcserver/serverinterceptors"
	"goshop/pkg/common/util/contextutil"
	"goshop/pkg/host"
	"goshop/pkg/log"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

type ServerOption func(o *Server)

var (
	errServerAlreadyStarted = errors.New("grpc server already started")
	errServerStopped        = errors.New("grpc server is stopped")
)

const defaultStopTimeout = 10 * time.Second

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
	registrars     []ServerRegistrar
	lifecycleMu    sync.Mutex
	endpoint       *url.URL
	started        bool
	stopped        bool
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
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.endpoint == nil {
		return nil
	}
	endpoint := *s.endpoint
	return &endpoint
}

func (s *Server) Address() string {
	return s.address
}

func (s *Server) Ready() <-chan struct{} {
	return s.ready
}

// NewServer builds a gRPC server without binding its listener. Configuration
// errors are returned to the caller; network binding happens in Start.
func NewServer(opts ...ServerOption) (*Server, error) {
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

	for _, registrar := range srv.registrars {
		registrar(srv.Server)
	}

	//注册health
	grpc_health_v1.RegisterHealthServer(srv.Server, srv.health)
	if srv.enableReflection {
		reflection.Register(srv.Server)
	}
	//可以支持用户直接通过grpc的一个接口查看当前支持的所有的rpc服务

	return srv, nil
}

// NewServerE is retained for source compatibility. Deprecated: use NewServer.
func NewServerE(opts ...ServerOption) (*Server, error) {
	return NewServer(opts...)
}

// ServerRegistrar registers an application-owned service on a gRPC server.
type ServerRegistrar func(*grpc.Server)

// WithRegistrar registers an application-owned gRPC service after the server
// is constructed. Framework code deliberately does not import application APIs.
func WithRegistrar(registrar ServerRegistrar) ServerOption {
	return func(s *Server) {
		if registrar != nil {
			s.registrars = append(s.registrars, registrar)
		}
	}
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

// WithLis supplies a listener whose ownership is transferred to Server.
// Stop closes it even when Start was never called.
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

// listenAndEndpoint creates the listener immediately before serving and
// records the endpoint that will be advertised through service discovery.
func (s *Server) listenAndEndpoint() error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.stopped {
		return errServerStopped
	}
	if s.started {
		return errServerAlreadyStarted
	}
	if s.lis == nil {
		lis, err := net.Listen("tcp", s.address)
		if err != nil {
			return fmt.Errorf("listen grpc server on %s: %w", s.address, err)
		}
		s.lis = lis
	}
	addr, err := host.Extract(s.address, s.lis)
	if err != nil {
		_ = s.lis.Close()
		s.lis = nil
		return fmt.Errorf("extract grpc endpoint: %w", err)
	}
	s.endpoint = discovery.NewEndpoint("grpc", addr, s.tlsEnabled)
	s.started = true
	return nil
}

// Start binds the configured address and begins serving gRPC requests.
func (s *Server) Start(ctx context.Context) error {
	ctx, releaseCtx := contextutil.OrProcess(ctx)
	defer releaseCtx()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.listenAndEndpoint(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultStopTimeout)
		stopErr := s.Stop(cleanupCtx)
		cancel()
		if stopErr != nil {
			return errors.Join(err, fmt.Errorf("stop grpc server after canceled start: %w", stopErr))
		}
		return err
	}
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
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = contextutil.NewOperation(defaultStopTimeout)
		defer cancel()
	}
	s.lifecycleMu.Lock()
	if s.stopped {
		s.lifecycleMu.Unlock()
		return nil
	}
	s.stopped = true
	started := s.started
	lis := s.lis
	if !started {
		s.lis = nil
	}
	s.lifecycleMu.Unlock()
	if !started {
		if lis != nil {
			if err := lis.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("close unstarted grpc listener: %w", err)
			}
		}
		return nil
	}
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
	if lis != nil {
		if err := lis.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("close grpc listener: %w", err)
		}
	}
	log.Infof("[grpc] server stopped")
	return nil
}
