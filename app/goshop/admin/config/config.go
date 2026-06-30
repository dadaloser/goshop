package config

import (
	"encoding/json"

	"goshop/app/pkg/options"
	"goshop/pkg/app"
	cliflag "goshop/pkg/common/cli/flag"
	"goshop/pkg/log"
)

type Config struct {
	Log *log.Options `json:"log" mapstructure:"log"`

	Server   *options.ServerOptions   `json:"server" mapstructure:"server"`
	Registry *options.RegistryOptions `json:"registry" mapstructure:"registry"`
}

func (c *Config) Validate() []error {
	var errors []error
	errors = append(errors, c.Log.Validate()...)
	errors = append(errors, c.Server.Validate()...)
	errors = append(errors, c.Registry.Validate()...)
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
	return fss
}

func New() *Config {
	//配置默认初始化
	return &Config{
		Log:      log.NewOptions(),
		Server:   options.NewServerOptions(),
		Registry: options.NewRegistryOptions(),
	}
}
