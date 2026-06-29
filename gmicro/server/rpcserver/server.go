package rpcserver

import (
	"context"

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

	address    string
	unaryInts  []grpc.UnaryServerInterceptor
	streamInts []grpc.StreamServerInterceptor
	grpcOpts   []grpc.ServerOption
	lis        net.Listener
	timeout    time.Duration

	health   *health.Server
	metadata *apimd.Server
	endpoint *url.URL

	enableMetrics    bool
	enableReflection bool
}

func (s *Server) Endpoint() *url.URL {
	return s.endpoint
}

func (s *Server) Address() string {
	return s.address
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
		address: ":0",
		health:  health.NewServer(),
		//timeout: 1 * time.Second,
	}

	for _, o := range opts {
		o(srv)
	}

	//TODO 我们现在希望用户不设置拦截器的情况下，我们会自动默认加上一些必须的拦截器， crash，tracing
	unaryInts := []grpc.UnaryServerInterceptor{
		srvintc.UnaryCrashInterceptor,
	}

	if srv.enableMetrics {
		unaryInts = append(unaryInts, srvintc.UnaryPrometheusInterceptor)
	}

	if srv.timeout > 0 {
		unaryInts = append(unaryInts, srvintc.UnaryTimeoutInterceptor(srv.timeout))
	}

	if len(srv.unaryInts) > 0 {
		unaryInts = append(unaryInts, srv.unaryInts...)
	}

	//把我们传入的拦截器转换成grpc的ServerOption
	grpcOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unaryInts...),
		//注意:链路追踪拦截器需要独立出来
		grpc.StatsHandler(otelgrpc.NewServerHandler())}
	if len(srv.streamInts) > 0 {
		grpcOpts = append(grpcOpts, grpc.ChainStreamInterceptor(srv.streamInts...))
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

func WithTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
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
		s.grpcOpts = append(s.grpcOpts,
			grpc.MaxConcurrentStreams(1024),
			grpc.KeepaliveParams(keepalive.ServerParameters{
				Time:    30 * time.Second,
				Timeout: 10 * time.Second,
			}),
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             10 * time.Second,
				PermitWithoutStream: true,
			}),
		)
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
	s.endpoint = &url.URL{Scheme: "grpc", Host: addr}
	return nil
}

// Start 启动grpc的服务
func (s *Server) Start(ctx context.Context) error {
	log.Infof("[grpc] server listening on: %s", s.lis.Addr().String())
	s.health.Resume()
	return s.Serve(s.lis)
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
