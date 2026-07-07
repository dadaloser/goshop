package config

import (
	"encoding/json"
	"goshop/app/pkg/options"
	"goshop/pkg/app"
	cliflag "goshop/pkg/common/cli/flag"
	"goshop/pkg/log"
)

type Config struct {
	MySQLOptions *options.MySQLOptions     `json:"mysql"     mapstructure:"mysql"`
	Log          *log.Options              `json:"log"     mapstructure:"log"`
	Server       *options.ServerOptions    `json:"server"     mapstructure:"server"`
	Telemetry    *options.TelemetryOptions `json:"telemetry" mapstructure:"telemetry"`
	Registry     *options.RegistryOptions  `json:"registry" mapstructure:"registry"`
	RedisOptions *options.RedisOptions     `json:"redis" mapstructure:"redis"` //涉及到分布式锁功能，后续会用到redis
}

func New() *Config {
	//配置默认初始化
	return &Config{
		MySQLOptions: options.NewMySQLOptions(),
		Log:          log.NewOptions(),
		Server:       options.NewServerOptions(),
		Telemetry:    options.NewTelemetryOptions(),
		Registry:     options.NewRegistryOptions(),
		RedisOptions: options.NewRedisOptions(),
	}
}

// Flags returns flags for a specific APIServer by section name.
func (o *Config) Flags() (fss cliflag.NamedFlagSets) {
	o.Server.AddFlags(fss.FlagSet("server"))
	o.Log.AddFlags(fss.FlagSet("logs"))
	o.Telemetry.AddFlags(fss.FlagSet("telemetry"))
	o.Registry.AddFlags(fss.FlagSet("registry"))
	o.MySQLOptions.AddFlags(fss.FlagSet("mysql"))
	o.RedisOptions.AddFlags(fss.FlagSet("redis"))

	return fss
}

func (o *Config) String() string {
	data, _ := json.Marshal(o)

	return string(data)
}

func (o *Config) SafeString() string {
	return app.RedactJSON(o.String())
}

func (o *Config) ValidateStartup() error {
	if o.Server != nil {
		if err := o.Server.ValidateStartup(); err != nil {
			return err
		}
	}
	if o.MySQLOptions != nil {
		if err := o.MySQLOptions.ValidateStartup(); err != nil {
			return err
		}
	}
	if o.RedisOptions != nil {
		if err := o.RedisOptions.ValidateStartup(); err != nil {
			return err
		}
	}
	return nil
}

func (o *Config) Validate() []error {
	var errs []error

	errs = append(errs, o.MySQLOptions.Validate()...)
	errs = append(errs, o.Log.Validate()...)
	errs = append(errs, o.Server.Validate()...)
	errs = append(errs, o.Telemetry.Validate()...)
	errs = append(errs, o.Registry.Validate()...)
	errs = append(errs, o.RedisOptions.Validate()...)
	return errs
}
