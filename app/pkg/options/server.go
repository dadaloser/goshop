package options

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/spf13/pflag"
)

type ServerOptions struct {
	//是否开启pprof
	EnableProfiling bool `json:"profiling"      mapstructure:"profiling"`

	ProfilingToken string `json:"profiling-token,omitempty" mapstructure:"profiling-token"`

	//限流器
	EnableLimit bool `json:"limit"      mapstructure:"limit"`

	RateLimitRPS          float64 `json:"rate-limit-rps,omitempty"             mapstructure:"rate-limit-rps"`
	RateLimitBurst        int     `json:"rate-limit-burst,omitempty"           mapstructure:"rate-limit-burst"`
	MaxConcurrentRequests int     `json:"max-concurrent-requests,omitempty"    mapstructure:"max-concurrent-requests"`

	//是否开启metrics
	EnableMetrics bool `json:"enable-metrics" mapstructure:"enable-metrics"`

	//是否开启health check
	EnableHealthCheck bool `json:"enable-health-check" mapstructure:"enable-health-check"`

	//host
	Host string `json:"host,omitempty"                     mapstructure:"host"`

	//port
	Port int `json:"port,omitempty"                     mapstructure:"port"`

	//http port
	HttpPort int `json:"http-port,omitempty"                     mapstructure:"http-port"`

	//名称
	Name string `json:"name,omitempty"                 mapstructure:"name"`

	//中间件
	Middlewares []string `json:"middlewares,omitempty"                 mapstructure:"middlewares"`

	CorsAllowOrigins []string `json:"cors-allow-origins,omitempty" mapstructure:"cors-allow-origins"`

	ReadHeaderTimeout time.Duration `json:"read-header-timeout,omitempty" mapstructure:"read-header-timeout"`
	ReadTimeout       time.Duration `json:"read-timeout,omitempty"        mapstructure:"read-timeout"`
	WriteTimeout      time.Duration `json:"write-timeout,omitempty"       mapstructure:"write-timeout"`
	IdleTimeout       time.Duration `json:"idle-timeout,omitempty"        mapstructure:"idle-timeout"`
}

// NewServerOptions create a `zero` value instance.
func NewServerOptions() *ServerOptions {
	return &ServerOptions{
		EnableHealthCheck:     true,
		EnableProfiling:       false, //
		ProfilingToken:        "",
		EnableLimit:           false,
		RateLimitRPS:          100,
		RateLimitBurst:        200,
		MaxConcurrentRequests: 200,
		EnableMetrics:         true,
		Host:                  "127.0.0.1",
		Port:                  8078,
		HttpPort:              8079,
		Name:                  "goshop-user-srv",
		ReadHeaderTimeout:     5 * time.Second,
		ReadTimeout:           15 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           60 * time.Second,
	}
}

// Validate verifies flags passed to ServerOptions.
func (so *ServerOptions) Validate() []error {
	errs := []error{}
	if so.Name == "" {
		errs = append(errs, fmt.Errorf("server.name is required"))
	}
	if so.Host == "" {
		errs = append(errs, fmt.Errorf("server.host is required"))
	}
	if so.Port <= 0 || so.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port must be between 1 and 65535, got %d", so.Port))
	}
	if so.HttpPort < 0 || so.HttpPort > 65535 {
		errs = append(errs, fmt.Errorf("server.http-port must be between 0 and 65535, got %d", so.HttpPort))
	}
	if so.EnableProfiling && so.ProfilingToken == "" {
		errs = append(errs, fmt.Errorf("server.profiling-token is required when profiling is enabled"))
	}
	if so.EnableLimit {
		if so.RateLimitRPS <= 0 {
			errs = append(errs, fmt.Errorf("server.rate-limit-rps must be positive when limit is enabled"))
		}
		if so.RateLimitBurst <= 0 {
			errs = append(errs, fmt.Errorf("server.rate-limit-burst must be positive when limit is enabled"))
		}
		if so.MaxConcurrentRequests <= 0 {
			errs = append(errs, fmt.Errorf("server.max-concurrent-requests must be positive when limit is enabled"))
		}
	}
	return errs
}

func (so *ServerOptions) ValidateStartup() error {
	if so.Host == "" {
		return errors.New("server.host is required")
	}
	if so.Port <= 0 || so.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", so.Port)
	}
	if so.HttpPort < 0 || so.HttpPort > 65535 {
		return fmt.Errorf("server.http-port must be between 0 and 65535, got %d", so.HttpPort)
	}
	if so.EnableProfiling && so.ProfilingToken == "" {
		return errors.New("server.profiling-token is required when profiling is enabled")
	}
	if so.ReadHeaderTimeout <= 0 || so.ReadTimeout <= 0 || so.WriteTimeout <= 0 || so.IdleTimeout <= 0 {
		return errors.New("server.read-header-timeout, server.read-timeout, server.write-timeout and server.idle-timeout must be positive")
	}
	if slices.Contains(so.Middlewares, "cors") {
		if len(so.CorsAllowOrigins) == 0 {
			return errors.New("server.cors-allow-origins is required when cors middleware is enabled")
		}
		for _, origin := range so.CorsAllowOrigins {
			if origin == "" || origin == "*" {
				return errors.New("server.cors-allow-origins must not contain empty or wildcard origins")
			}
		}
	}

	return nil
}

// AddFlags adds flags related to server storage for a specific APIServer to the specified FlagSet.
func (so *ServerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&so.EnableProfiling, "server.enable-profiling", so.EnableProfiling,
		"enable-profiling, if true, will add <host>:<port>/debug/pprof/, default is false")
	fs.StringVar(&so.ProfilingToken, "server.profiling-token", so.ProfilingToken,
		"bearer token required to access /debug/pprof when profiling is enabled")
	fs.BoolVar(&so.EnableLimit, "server.enable-limit", so.EnableLimit,
		"enable server overload protection with rate and concurrency limiters")
	fs.Float64Var(&so.RateLimitRPS, "server.rate-limit-rps", so.RateLimitRPS,
		"maximum accepted REST requests per second when limit is enabled")
	fs.IntVar(&so.RateLimitBurst, "server.rate-limit-burst", so.RateLimitBurst,
		"maximum REST rate limiter burst when limit is enabled")
	fs.IntVar(&so.MaxConcurrentRequests, "server.max-concurrent-requests", so.MaxConcurrentRequests,
		"maximum concurrent REST requests when limit is enabled")
	fs.BoolVar(&so.EnableMetrics, "server.enable-metrics", so.EnableMetrics,
		"enable-metrics, if true, will add /metrics, default is true")

	fs.BoolVar(&so.EnableHealthCheck, "server.enable-health-check", so.EnableHealthCheck,
		"enable-health-check, if true, will add health check route, default is true")

	fs.StringVar(&so.Host, "server.host", so.Host, "server host default is 127.0.0.1")

	fs.IntVar(&so.Port, "server.port", so.Port, "server port default is 8078")

	fs.IntVar(&so.HttpPort, "server.http-port", so.HttpPort, "server http port default is 8079")

	fs.StringVar(&so.Name, "server.name", so.Name, "server name default is goshop-user-srv")
	fs.StringSliceVar(&so.CorsAllowOrigins, "server.cors-allow-origins", so.CorsAllowOrigins,
		"allowed CORS origins for production when cors middleware is enabled")

	fs.DurationVar(&so.ReadHeaderTimeout, "server.read-header-timeout", so.ReadHeaderTimeout,
		"maximum duration for reading request headers")
	fs.DurationVar(&so.ReadTimeout, "server.read-timeout", so.ReadTimeout,
		"maximum duration for reading the entire request")
	fs.DurationVar(&so.WriteTimeout, "server.write-timeout", so.WriteTimeout,
		"maximum duration before timing out writes of the response")
	fs.DurationVar(&so.IdleTimeout, "server.idle-timeout", so.IdleTimeout,
		"maximum amount of time to wait for the next request when keep-alives are enabled")
}
