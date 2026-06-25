package restserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/penglongli/gin-metrics/ginmetrics"

	mws "goshop/gmicro/server/restserver/middlewares"
	"goshop/gmicro/server/restserver/pprof"
	"goshop/gmicro/server/restserver/validation"
	"goshop/pkg/errors"
	"goshop/pkg/host"
	"goshop/pkg/log"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
)

type JwtInfo struct {
	// defaults to "JWT"
	Realm string
	// defaults to empty
	Key string
	// defaults to 7 days
	Timeout time.Duration
	// defaults to 7 days
	MaxRefresh time.Duration
}

// wrapper for gin.Engine
type Server struct {
	*gin.Engine

	//端口号， 默认值 8080
	port int

	//监听地址，默认空字符串表示监听所有网卡
	host string

	//开发模式， 默认值 debug
	mode string

	//是否开启健康检查接口， 默认开启， 如果开启会自动添加 /health 接口
	healthCheck bool

	//是否开启pprof接口， 默认开启， 如果开启会自动添加 /debug/pprof 接口
	enableProfiling bool

	//是否开启metrics接口， 默认开启， 如果开启会自动添加 /metrics 接口
	enableMetrics bool

	//中间件
	middlewares []string

	//jwt配置信息
	jwt *JwtInfo

	//翻译器, 默认值 zh
	transName string
	trans     ut.Translator

	server *http.Server

	serviceName string

	ready     chan struct{}
	readyOnce sync.Once
	endpoint  *url.URL
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		port:            8080,
		mode:            "debug",
		healthCheck:     true,
		enableProfiling: true,
		jwt: &JwtInfo{
			"JWT",
			"mwGDMGtSpdwXaiihF5WnEgRajSFpdZj8",
			7 * 24 * time.Hour,
			7 * 24 * time.Hour,
		},
		Engine:      gin.Default(),
		transName:   "zh",
		serviceName: "gmicro",
		ready:       make(chan struct{}),
	}

	for _, o := range opts {
		o(srv)
	}

	srv.Use(mws.TracingHandler(srv.serviceName))
	for _, m := range srv.middlewares {
		mw, ok := mws.Middlewares[m]
		if !ok {
			log.Warnf("can not find middleware: %s", m)
			continue
			//panic(errors.Errorf("can not find middleware: %s", m))
		}

		log.Infof("intall middleware: %s", m)
		srv.Use(mw)
	}

	return srv
}

func (s *Server) Translator() ut.Translator {
	return s.trans
}

func (s *Server) Endpoint() *url.URL {
	return s.endpoint
}

func (s *Server) Address() string {
	return net.JoinHostPort(s.host, fmt.Sprintf("%d", s.port))
}

func (s *Server) Ready() <-chan struct{} {
	return s.ready
}

// Start  rest server
func (s *Server) Start(ctx context.Context) error {
	//设置开发模式，打印路由信息
	if s.mode != gin.DebugMode && s.mode != gin.ReleaseMode && s.mode != gin.TestMode {
		return errors.New("mode must be one of debug/release/test")
	}

	//设置开发模式，打印路由信息
	gin.SetMode(s.mode)
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		log.Infof("%-6s %-s --> %s(%d handlers)", httpMethod, absolutePath, handlerName, nuHandlers)
	}

	//TODO 初始化翻译器
	err := s.initTrans(s.transName)
	if err != nil {
		log.Errorf("initTrans error %s", err.Error())
		return err
	}

	//注册mobile验证码
	validation.RegisterMobile(s.trans)

	//根据配置初始化pprof路由
	if s.enableProfiling {
		pprof.Register(s.Engine)
	}

	if s.enableMetrics {
		// get global Monitor object
		m := ginmetrics.GetMonitor()
		// +optional set metric path, default /debug/metrics
		m.SetMetricPath("/metrics")
		// +optional set slow time, default 5s
		// +optional set request duration, default {0.1, 0.3, 1.2, 5, 10}
		// used to p95, p99
		m.SetDuration([]float64{0.1, 0.3, 1.2, 5, 10})
		m.Use(s)
	}

	address := s.Address()
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	addr, err := host.Extract(address, lis)
	if err != nil {
		_ = lis.Close()
		return err
	}
	s.endpoint = &url.URL{Scheme: "http", Host: addr}

	log.Infof("rest server is running on: %s", lis.Addr().String())
	s.server = &http.Server{
		Handler: s.Engine,
	}
	_ = s.SetTrustedProxies(nil)
	s.readyOnce.Do(func() {
		close(s.ready)
	})
	if err = s.server.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Infof("rest server is stopping")
	if s.server == nil {
		log.Info("rest server stopped")
		return nil
	}
	if err := s.server.Shutdown(ctx); err != nil {
		log.Errorf("rest server shutdown error: %s", err.Error())
		return err
	}
	log.Info("rest server stopped")
	return nil
}
