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
	MySQLOptions        *options.MySQLOptions       `json:"mysql"     mapstructure:"mysql"`
	Log                 *log.Options                `json:"log"     mapstructure:"log"`
	Server              *options.ServerOptions      `json:"server"     mapstructure:"server"`
	Telemetry           *options.TelemetryOptions   `json:"telemetry" mapstructure:"telemetry"`
	Registry            *options.RegistryOptions    `json:"registry" mapstructure:"registry"`
	RPC                 *options.RPCSecurityOptions `json:"rpc-security" mapstructure:"rpc-security"`
	Dtm                 *options.DtmOptions         `json:"dtm" mapstructure:"dtm"`
	Lifecycle           *LifecycleOptions           `json:"lifecycle" mapstructure:"lifecycle"`
	RPCClientResilience *resilience.Options         `json:"rpc-client-resilience" mapstructure:"rpc-client-resilience"`
}

func New() *Config {
	//配置默认初始化
	return &Config{
		MySQLOptions:        options.NewMySQLOptions(),
		Log:                 log.NewOptions(),
		Server:              options.NewServerOptions(),
		Telemetry:           options.NewTelemetryOptions(),
		Registry:            options.NewRegistryOptions(),
		RPC:                 options.NewRPCSecurityOptions(),
		Dtm:                 options.NewDtmOptions(),
		Lifecycle:           NewLifecycleOptions(),
		RPCClientResilience: resilience.NewOptions(),
	}
}

// Flags returns flags for a specific APIServer by section name.
func (o *Config) Flags() (fss cliflag.NamedFlagSets) {
	o.Server.AddFlags(fss.FlagSet("server"))
	o.Log.AddFlags(fss.FlagSet("logs"))
	o.Telemetry.AddFlags(fss.FlagSet("telemetry"))
	o.Registry.AddFlags(fss.FlagSet("registry"))
	o.RPC.AddFlags(fss.FlagSet("rpc-security"))
	o.MySQLOptions.AddFlags(fss.FlagSet("mysql"))
	o.Dtm.AddFlags(fss.FlagSet("dtm"))
	o.Lifecycle.AddFlags(fss.FlagSet("lifecycle"))
	o.RPCClientResilience.AddFlags(fss.FlagSet("rpc-client-resilience"), "rpc-client-resilience")
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
		if err := o.RPC.ValidateStartup(); err != nil {
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
	errs = append(errs, o.Dtm.Validate()...)
	errs = append(errs, o.Lifecycle.Validate()...)
	errs = append(errs, o.RPCClientResilience.Validate()...)
	return errs
}
