package restserver

import (
	"time"

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
