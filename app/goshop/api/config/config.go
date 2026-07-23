package config

import (
	"encoding/json"

	"goshop/app/pkg/options"
	"goshop/gmicro/resilience"
	"goshop/pkg/app"
	cliflag "goshop/pkg/common/cli/flag"
	"goshop/pkg/log"
)

type Config struct {
	Log *log.Options `json:"log" mapstructure:"log"`

	Server              *options.ServerOptions      `json:"server" mapstructure:"server"`
	Registry            *options.RegistryOptions    `json:"registry" mapstructure:"registry"`
	RPC                 *options.RPCSecurityOptions `json:"rpc-security" mapstructure:"rpc-security"`
	Jwt                 *options.JwtOptions         `json:"jwt" mapstructure:"jwt"`
	Sms                 *options.SmsOptions         `json:"sms" mapstructure:"sms"`
	Email               *options.EmailOptions       `json:"email" mapstructure:"email"`
	Payment             *options.PaymentOptions     `json:"payment" mapstructure:"payment"`
	Redis               *options.RedisOptions       `json:"redis" mapstructure:"redis"`
	RPCClientResilience *resilience.Options         `json:"rpc-client-resilience" mapstructure:"rpc-client-resilience"`
}

func (c *Config) Validate() []error {
	var errors []error
	errors = append(errors, c.Log.Validate()...)
	errors = append(errors, c.Server.Validate()...)
	errors = append(errors, c.Registry.Validate()...)
	errors = append(errors, c.RPC.Validate()...)
	errors = append(errors, c.Jwt.Validate()...)
	errors = append(errors, c.Sms.Validate()...)
	errors = append(errors, c.Email.Validate()...)
	errors = append(errors, c.Payment.Validate()...)
	errors = append(errors, c.Redis.Validate()...)
	errors = append(errors, c.RPCClientResilience.Validate()...)
	return errors
}

func (c *Config) ValidateStartup() error {
	if c.Server != nil {
		if err := c.Server.ValidateStartup(); err != nil {
			return err
		}
	}
	if c.Jwt != nil {
		if err := c.Jwt.ValidateStartup(); err != nil {
			return err
		}
	}
	if c.RPC != nil {
		if err := c.RPC.ValidateStartup(); err != nil {
			return err
		}
	}
	if c.Redis != nil {
		if err := c.Redis.ValidateStartup(); err != nil {
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
	c.RPC.AddFlags(fss.FlagSet("rpc-security"))
	c.Jwt.AddFlags(fss.FlagSet("jwt"))
	c.Sms.AddFlags(fss.FlagSet("sms"))
	c.Email.AddFlags(fss.FlagSet("email"))
	c.Payment.AddFlags(fss.FlagSet("payment"))
	c.Redis.AddFlags(fss.FlagSet("redis"))
	c.RPCClientResilience.AddFlags(fss.FlagSet("rpc-client-resilience"), "rpc-client-resilience")
	return fss
}

func New() *Config {
	//配置默认初始化
	return &Config{
		Log:                 log.NewOptions(),
		Server:              options.NewServerOptions(),
		Registry:            options.NewRegistryOptions(),
		RPC:                 options.NewRPCSecurityOptions(),
		Jwt:                 options.NewJwtOptions(),
		Sms:                 options.NewSmsOptions(),
		Email:               options.NewEmailOptions(),
		Payment:             options.NewPaymentOptions(),
		Redis:               options.NewRedisOptions(),
		RPCClientResilience: resilience.NewOptions(),
	}
}
