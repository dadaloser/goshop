package restserver

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	mws "goshop/gmicro/server/restserver/middlewares"
)

type ServerOption func(*Server)

func WithEnableProfiling(profiling bool) ServerOption {
	return func(s *Server) {
		s.enableProfiling = profiling
	}
}

func WithProfilingToken(token string) ServerOption {
	return func(s *Server) {
		s.profilingToken = token
	}
}

func WithBuiltInRouteCIDRs(cidrs []string) ServerOption {
	return func(s *Server) {
		if len(cidrs) == 0 {
			return
		}
		nets, err := withBuiltInRouteCIDRs(cidrs)
		if err != nil {
			s.builtInRouteErr = err
			return
		}
		s.builtInRouteCIDRs = nets
	}
}

func WithMode(mode string) ServerOption {
	return func(s *Server) {
		s.mode = mode
	}
}

func WithServiceName(srvName string) ServerOption {
	return func(s *Server) {
		s.serviceName = srvName
	}
}

func WithPort(port int) ServerOption {
	return func(s *Server) {
		s.port = port
	}
}

func WithHost(host string) ServerOption {
	return func(s *Server) {
		s.host = host
	}
}

func WithMiddlewares(middlewares []string) ServerOption {
	return func(s *Server) {
		s.middlewares = middlewares
	}
}

// WithNamedMiddleware registers middleware on one server instance. It avoids
// process-global mutable registries and makes extension ownership explicit.
// Add name to WithMiddlewares to control its position in the configured chain.
func WithNamedMiddleware(name string, middleware gin.HandlerFunc) ServerOption {
	return func(s *Server) {
		if name == "" || middleware == nil {
			s.middlewareConfigErr = fmt.Errorf("named middleware requires a non-empty name and handler")
			return
		}
		if name == "recovery" || name == "logger" || name == "cors" {
			s.middlewareConfigErr = fmt.Errorf("middleware name %q is reserved", name)
			return
		}
		if _, exists := s.customMiddlewares[name]; exists {
			s.middlewareConfigErr = fmt.Errorf("duplicate named middleware %q", name)
			return
		}
		if _, builtIn := mws.Lookup(name); builtIn {
			s.middlewareConfigErr = fmt.Errorf("middleware name %q is built in", name)
			return
		}
		s.customMiddlewares[name] = middleware
	}
}

func WithCorsOptions(opts mws.CorsOptions) ServerOption {
	return func(s *Server) {
		s.corsOptions = &opts
	}
}

func WithHealthCheck(health bool) ServerOption {
	return func(s *Server) {
		s.healthCheck = health
	}
}

// WithReadinessCheck adds a dependency check evaluated by /readyz and /healthz.
func WithReadinessCheck(check func() error) ServerOption {
	return func(s *Server) {
		if check != nil {
			s.readinessChecks = append(s.readinessChecks, check)
		}
	}
}

func WithJwt(jwt *JwtInfo) ServerOption {
	return func(s *Server) {
		s.jwt = jwt
		s.requireJWTKey = true
	}
}

func WithTransNames(transName string) ServerOption {
	return func(s *Server) {
		s.transName = transName
	}
}

func WithMetrics(enable bool) ServerOption {
	return func(o *Server) {
		o.enableMetrics = enable
	}
}

func WithReadHeaderTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.readHeaderTimeout = timeout
	}
}

func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.readTimeout = timeout
	}
}

func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.writeTimeout = timeout
	}
}

func WithIdleTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.idleTimeout = timeout
	}
}

func WithRateLimit(rps float64, burst int) ServerOption {
	return func(s *Server) {
		if rps > 0 && burst > 0 {
			s.rateLimit = rate.Limit(rps)
			s.rateLimitBurst = burst
		}
	}
}

// WithClientRouteRateLimit limits each client IP independently for each HTTP
// method and route template. maxKeys bounds the in-memory limiter cache.
func WithClientRouteRateLimit(rps float64, burst, maxKeys int) ServerOption {
	return func(s *Server) {
		if rps > 0 && burst > 0 && maxKeys > 0 {
			s.clientRateLimit = rate.Limit(rps)
			s.clientRateLimitBurst = burst
			s.clientRateLimitMaxKeys = maxKeys
		}
	}
}

func WithMaxConcurrentRequests(limit int) ServerOption {
	return func(s *Server) {
		if limit > 0 {
			s.maxConcurrentReqs = limit
		}
	}
}

func WithStartupValidator(validate StartupValidator) ServerOption {
	return func(s *Server) {
		if validate != nil {
			s.startupValidators = append(s.startupValidators, validate)
		}
	}
}
