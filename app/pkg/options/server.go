package options

import (
	"time"

	"github.com/spf13/pflag"
)

type ServerOptions struct {
	//是否开启pprof
	EnableProfiling bool `json:"profiling"      mapstructure:"profiling"`

	//限流器
	EnableLimit bool `json:"limit"      mapstructure:"limit"`

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

	ReadHeaderTimeout time.Duration `json:"read-header-timeout,omitempty" mapstructure:"read-header-timeout"`
	ReadTimeout       time.Duration `json:"read-timeout,omitempty"        mapstructure:"read-timeout"`
	WriteTimeout      time.Duration `json:"write-timeout,omitempty"       mapstructure:"write-timeout"`
	IdleTimeout       time.Duration `json:"idle-timeout,omitempty"        mapstructure:"idle-timeout"`
}

// NewServerOptions create a `zero` value instance.
func NewServerOptions() *ServerOptions {
	return &ServerOptions{
		EnableHealthCheck: true,
		EnableProfiling:   false, //
		EnableMetrics:     true,
		Host:              "127.0.0.1",
		Port:              8078,
		HttpPort:          8079,
		Name:              "goshop-user-srv",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// Validate verifies flags passed to ServerOptions.
func (so *ServerOptions) Validate() []error {
	errs := []error{}
	return errs
}

// AddFlags adds flags related to server storage for a specific APIServer to the specified FlagSet.
func (so *ServerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&so.EnableProfiling, "server.enable-profiling", so.EnableProfiling,
		"enable-profiling, if true, will add <host>:<port>/debug/pprof/, default is false")
	fs.BoolVar(&so.EnableMetrics, "server.enable-metrics", so.EnableMetrics,
		"enable-metrics, if true, will add /metrics, default is true")

	fs.BoolVar(&so.EnableHealthCheck, "server.enable-health-check", so.EnableHealthCheck,
		"enable-health-check, if true, will add health check route, default is true")

	fs.StringVar(&so.Host, "server.host", so.Host, "server host default is 127.0.0.1")

	fs.IntVar(&so.Port, "server.port", so.Port, "server port default is 8078")

	fs.IntVar(&so.HttpPort, "server.http-port", so.HttpPort, "server http port default is 8079")

	fs.StringVar(&so.Name, "server.name", so.Name, "server name default is goshop-user-srv")

	fs.DurationVar(&so.ReadHeaderTimeout, "server.read-header-timeout", so.ReadHeaderTimeout,
		"maximum duration for reading request headers")
	fs.DurationVar(&so.ReadTimeout, "server.read-timeout", so.ReadTimeout,
		"maximum duration for reading the entire request")
	fs.DurationVar(&so.WriteTimeout, "server.write-timeout", so.WriteTimeout,
		"maximum duration before timing out writes of the response")
	fs.DurationVar(&so.IdleTimeout, "server.idle-timeout", so.IdleTimeout,
		"maximum amount of time to wait for the next request when keep-alives are enabled")
}
