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
	MySQLOptions        *options.MySQLOptions       `json:"mysql" mapstructure:"mysql"`
	Log                 *log.Options                `json:"log" mapstructure:"log"`
	Server              *options.ServerOptions      `json:"server" mapstructure:"server"`
	Telemetry           *options.TelemetryOptions   `json:"telemetry" mapstructure:"telemetry"`
	Registry            *options.RegistryOptions    `json:"registry" mapstructure:"registry"`
	RPC                 *options.RPCSecurityOptions `json:"rpc-security" mapstructure:"rpc-security"`
	RPCClientResilience *resilience.Options         `json:"rpc-client-resilience" mapstructure:"rpc-client-resilience"`
	Outbox              *OutboxOptions              `json:"outbox" mapstructure:"outbox"`
}

func New() *Config {
	return &Config{
		MySQLOptions:        options.NewMySQLOptions(),
		Log:                 log.NewOptions(),
		Server:              options.NewServerOptions(),
		Telemetry:           options.NewTelemetryOptions(),
		Registry:            options.NewRegistryOptions(),
		RPC:                 options.NewRPCSecurityOptions(),
		RPCClientResilience: resilience.NewOptions(),
		Outbox:              NewOutboxOptions(),
	}
}

func (o *Config) Flags() (fss cliflag.NamedFlagSets) {
	o.Server.AddFlags(fss.FlagSet("server"))
	o.Log.AddFlags(fss.FlagSet("logs"))
	o.Telemetry.AddFlags(fss.FlagSet("telemetry"))
	o.Registry.AddFlags(fss.FlagSet("registry"))
	o.RPC.AddFlags(fss.FlagSet("rpc-security"))
	o.MySQLOptions.AddFlags(fss.FlagSet("mysql"))
	o.RPCClientResilience.AddFlags(fss.FlagSet("rpc-client-resilience"), "rpc-client-resilience")
	o.Outbox.AddFlags(fss.FlagSet("outbox"))
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
	if o.RPC != nil {
		if err := o.RPC.ValidateServerStartup(); err != nil {
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
	errs = append(errs, o.RPC.Validate()...)
	errs = append(errs, o.RPCClientResilience.Validate()...)
	errs = append(errs, o.Outbox.Validate()...)
	return errs
}
