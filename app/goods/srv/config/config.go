package config

import (
	"encoding/json"

	"goshop/app/pkg/options"
	"goshop/pkg/app"
	cliflag "goshop/pkg/common/cli/flag"
	"goshop/pkg/log"
)

type Config struct {
	Log       *log.Options       `json:"log" mapstructure:"log"`
	EsOptions *options.EsOptions `json:"es" mapstructure:"es"`

	Server       *options.ServerOptions    `json:"server" mapstructure:"server"`
	Registry     *options.RegistryOptions  `json:"registry" mapstructure:"registry"`
	Telemetry    *options.TelemetryOptions `json:"telemetry" mapstructure:"telemetry"`
	MySQLOptions *options.MySQLOptions     `json:"mysql" mapstructure:"mysql"`
}

func (c *Config) Validate() []error {
	var errors []error
	errors = append(errors, c.Log.Validate()...)
	errors = append(errors, c.Server.Validate()...)
	errors = append(errors, c.Registry.Validate()...)
	errors = append(errors, c.Telemetry.Validate()...)
	errors = append(errors, c.MySQLOptions.Validate()...)
	errors = append(errors, c.EsOptions.Validate()...)
	return errors
}

func (c *Config) ValidateStartup() error {
	if c.Log != nil && c.Log.Development {
		return nil
	}
	if c.Server != nil {
		if err := c.Server.ValidateStartup(); err != nil {
			return err
		}
	}
	if c.MySQLOptions != nil {
		if err := c.MySQLOptions.ValidateStartup(); err != nil {
			return err
		}
	}
	if c.EsOptions != nil {
		if err := c.EsOptions.ValidateStartup(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) String() string {
	data, _ := json.Marshal(c)

	return string(data)
}

func (c *Config) SafeString() string {
	return app.RedactJSON(c.String())
}

func (c *Config) Flags() (fss cliflag.NamedFlagSets) {
	c.Log.AddFlags(fss.FlagSet("logs"))
	c.Server.AddFlags(fss.FlagSet("server"))
	c.Registry.AddFlags(fss.FlagSet("registry"))
	c.Telemetry.AddFlags(fss.FlagSet("telemetry"))
	c.MySQLOptions.AddFlags(fss.FlagSet("mysql"))
	c.EsOptions.AddFlags(fss.FlagSet("es"))
	return fss
}

func New() *Config {
	//配置默认初始化
	return &Config{
		Log:          log.NewOptions(),
		Server:       options.NewServerOptions(),
		Registry:     options.NewRegistryOptions(),
		Telemetry:    options.NewTelemetryOptions(),
		MySQLOptions: options.NewMySQLOptions(),
		EsOptions:    options.NewEsOptions(),
	}
}
