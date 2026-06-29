package restserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/penglongli/gin-metrics/ginmetrics"
	"golang.org/x/time/rate"

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

// StartupValidator validates server configuration before the listener starts.
type StartupValidator func(*Server) error

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

	readHeaderTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	startupValidators []StartupValidator
	rateLimit         rate.Limit
	rateLimitBurst    int
	maxConcurrentReqs int
	profilingToken    string

	//中间件
	middlewares []string
	corsOptions *mws.CorsOptions

	//jwt配置信息
	jwt           *JwtInfo
	requireJWTKey bool

	//翻译器, 默认值 zh
	transName string
	trans     ut.Translator

	server *http.Server

	serviceName string

	ready     chan struct{}
	readyOnce sync.Once
	draining  atomic.Bool
	endpoint  *url.URL
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		port:              8080,
		mode:              gin.ReleaseMode,
		healthCheck:       true,
		enableProfiling:   false,
		readHeaderTimeout: 5 * time.Second,
		readTimeout:       15 * time.Second,
		writeTimeout:      30 * time.Second,
		idleTimeout:       60 * time.Second,
		jwt: &JwtInfo{
			"JWT",
			"",
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
	if srv.maxConcurrentReqs > 0 {
		srv.Use(maxConcurrentRequestsMiddleware(srv.maxConcurrentReqs))
	}
	if srv.rateLimit > 0 && srv.rateLimitBurst > 0 {
		srv.Use(rateLimitMiddleware(rate.NewLimiter(srv.rateLimit, srv.rateLimitBurst)))
	}
	for _, m := range srv.middlewares {
		mw, ok := srv.middleware(m)
		if !ok {
			log.Warnf("can not find middleware: %s", m)
			continue
		}

		log.Infof("intall middleware: %s", m)
		srv.Use(mw)
	}

	return srv
}

func (s *Server) middleware(name string) (gin.HandlerFunc, bool) {
	if name == "cors" && s.corsOptions != nil {
		return mws.CorsWithOptions(*s.corsOptions), true
	}
	mw, ok := mws.Middlewares[name]
	return mw, ok
}

// ValidateStartupConfig validates server configuration before startup.
func (s *Server) ValidateStartupConfig() error {
	if s.mode != gin.DebugMode && s.mode != gin.ReleaseMode && s.mode != gin.TestMode {
		return errors.New("mode must be one of debug/release/test")
	}
	if s.mode == gin.ReleaseMode {
		if err := s.validateProductionConfig(); err != nil {
			return err
		}
	}
	for _, validate := range s.startupValidators {
		if validate == nil {
			continue
		}
		if err := validate(s); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) validateProductionConfig() error {
	if s.mode == gin.DebugMode {
		return errors.New("production rest server must not run in debug mode")
	}
	if s.enableProfiling {
		if s.profilingToken == "" {
			return errors.New("production rest server profiling requires explicit bearer token")
		}
	}
	if s.readHeaderTimeout <= 0 || s.readTimeout <= 0 || s.writeTimeout <= 0 || s.idleTimeout <= 0 {
		return errors.New("production rest server requires positive http timeouts")
	}
	if s.requireJWTKey && (s.jwt == nil || s.jwt.Key == "") {
		return errors.New("production rest server requires explicit jwt key")
	}
	if slices.Contains(s.middlewares, "cors") && !hasProductionCorsOrigins(s.corsOptions) {
		return errors.New("production rest server requires explicit cors allow origins")
	}
	return nil
}

func hasProductionCorsOrigins(opts *mws.CorsOptions) bool {
	if opts == nil || len(opts.AllowOrigins) == 0 {
		return false
	}
	for _, origin := range opts.AllowOrigins {
		if origin == "*" || origin == "" {
			return false
		}
	}
	return true
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
	s.draining.Store(false)
	if err := s.ValidateStartupConfig(); err != nil {
		return err
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

	//健康检查
	if s.healthCheck {
		s.registerHealthRoutes()
	}

	//根据配置初始化pprof路由
	if s.enableProfiling {
		s.registerProfilingRoutes()
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
		Handler:           s.Engine,
		ReadHeaderTimeout: s.readHeaderTimeout,
		ReadTimeout:       s.readTimeout,
		WriteTimeout:      s.writeTimeout,
		IdleTimeout:       s.idleTimeout,
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

func (s *Server) registerProfilingRoutes() {
	if s.profilingToken == "" {
		pprof.Register(s.Engine)
		return
	}
	pprof.RegisterWithMiddleware(s.Engine, bearerTokenMiddleware(s.profilingToken))
}

func (s *Server) Stop(ctx context.Context) error {
	log.Infof("rest server is stopping")
	s.draining.Store(true)
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

func bearerTokenMiddleware(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer "+token {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

func rateLimitMiddleware(limiter *rate.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}

func maxConcurrentRequestsMiddleware(limit int) gin.HandlerFunc {
	sem := make(chan struct{}, limit)
	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		default:
			c.AbortWithStatus(http.StatusServiceUnavailable)
		}
	}
}

func (s *Server) registerHealthRoutes() {
	livez := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
	readyz := func(c *gin.Context) {
		select {
		case <-s.ready:
			if s.draining.Load() {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
		default:
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
		}
	}

	s.GET("/livez", livez)
	s.GET("/readyz", readyz)
	s.GET("/healthz", readyz)
}
