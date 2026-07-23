package config

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"goshop/app/pkg/options"
	"goshop/pkg/app"
	cliflag "goshop/pkg/common/cli/flag"
	"goshop/pkg/log"

	"github.com/spf13/pflag"
)

type Config struct {
	Log *log.Options `json:"log" mapstructure:"log"`

	Server    *options.ServerOptions      `json:"server" mapstructure:"server"`
	Registry  *options.RegistryOptions    `json:"registry" mapstructure:"registry"`
	RPC       *options.RPCSecurityOptions `json:"rpc-security" mapstructure:"rpc-security"`
	Jwt       *options.JwtOptions         `json:"jwt" mapstructure:"jwt"`
	Redis     *options.RedisOptions       `json:"redis" mapstructure:"redis"`
	AdminAuth *AdminAuthOptions           `json:"admin-auth" mapstructure:"admin-auth"`
}

type AdminAuthOptions struct {
	Token                  string        `json:"token" mapstructure:"token"`
	ConfirmationToken      string        `json:"confirmation-token" mapstructure:"confirmation-token"`
	PreviousToken          string        `json:"-" mapstructure:"previous-token"`
	PreviousTokenExpiresAt time.Time     `json:"previous-token-expires-at" mapstructure:"previous-token-expires-at"`
	BreakGlassTTL          time.Duration `json:"break-glass-ttl" mapstructure:"break-glass-ttl"`
	BreakGlassKeyID        string        `json:"break-glass-key-id" mapstructure:"break-glass-key-id"`
}

func NewAdminAuthOptions() *AdminAuthOptions {
	return &AdminAuthOptions{}
}

func (o *AdminAuthOptions) EffectiveToken() string {
	if o != nil && o.Token != "" {
		return o.Token
	}
	return os.Getenv("GOSHOP_ADMIN_TOKEN")
}

func (o *AdminAuthOptions) EffectivePreviousToken() string {
	if o != nil && o.PreviousToken != "" {
		return o.PreviousToken
	}
	return os.Getenv("GOSHOP_ADMIN_PREVIOUS_TOKEN")
}
func (o *AdminAuthOptions) PreviousTokenActive(now time.Time) bool {
	return o != nil && o.EffectivePreviousToken() != "" && !o.PreviousTokenExpiresAt.IsZero() && now.Before(o.PreviousTokenExpiresAt)
}
func (o *AdminAuthOptions) EffectiveBreakGlassTTL() time.Duration {
	if o != nil && o.BreakGlassTTL > 0 && o.BreakGlassTTL <= 15*time.Minute {
		return o.BreakGlassTTL
	}
	return 15 * time.Minute
}
func (o *AdminAuthOptions) EffectiveBreakGlassKeyID() string {
	if o != nil && strings.TrimSpace(o.BreakGlassKeyID) != "" {
		return strings.TrimSpace(o.BreakGlassKeyID)
	}
	return "default"
}

func (o *AdminAuthOptions) EffectiveConfirmationToken() string {
	if o != nil && o.ConfirmationToken != "" {
		return o.ConfirmationToken
	}
	return os.Getenv("GOSHOP_ADMIN_CONFIRMATION_TOKEN")
}

func (o *AdminAuthOptions) Validate() []error {
	return nil
}

func (o *AdminAuthOptions) ValidateStartup() error {
	if o.EffectiveToken() == "" {
		return errors.New("admin-auth.token or GOSHOP_ADMIN_TOKEN is required")
	}
	if o.BreakGlassTTL < 0 || o.BreakGlassTTL > 15*time.Minute {
		return errors.New("admin-auth.break-glass-ttl must be between 0 and 15m")
	}
	if o.EffectivePreviousToken() != "" && o.PreviousTokenExpiresAt.IsZero() {
		return errors.New("admin-auth.previous-token-expires-at is required during rotation")
	}
	if adminTokenEqualForConfig(o.EffectiveToken(), o.EffectivePreviousToken()) {
		return errors.New("admin-auth.previous-token must differ from current token")
	}
	return nil
}

func adminTokenEqualForConfig(left, right string) bool {
	return left != "" && right != "" && left == right
}

func (o *AdminAuthOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		return
	}
	fs.StringVar(&o.Token, "admin-auth.token", o.Token, "break-glass token used only to issue a short-lived emergency identity without RBAC grants")
	fs.StringVar(&o.ConfirmationToken, "admin-auth.confirmation-token", o.ConfirmationToken, "second confirmation token required by high-risk admin write APIs")
	fs.StringVar(&o.PreviousToken, "admin-auth.previous-token", o.PreviousToken, "previous break-glass token accepted only during rotation overlap")
	fs.DurationVar(&o.BreakGlassTTL, "admin-auth.break-glass-ttl", o.BreakGlassTTL, "break-glass session TTL, maximum 15m")
	fs.StringVar(&o.BreakGlassKeyID, "admin-auth.break-glass-key-id", o.BreakGlassKeyID, "break-glass rotation key identifier")
}

func (c *Config) Validate() []error {
	var errors []error
	errors = append(errors, c.Log.Validate()...)
	errors = append(errors, c.Server.Validate()...)
	errors = append(errors, c.Registry.Validate()...)
	errors = append(errors, c.RPC.Validate()...)
	errors = append(errors, c.Jwt.Validate()...)
	errors = append(errors, c.Redis.Validate()...)
	errors = append(errors, c.AdminAuth.Validate()...)
	return errors
}

func (c *Config) ValidateStartup() error {
	if c.Server != nil {
		if err := c.Server.ValidateStartup(); err != nil {
			return err
		}
	}
	if c.AdminAuth == nil {
		return errors.New("admin-auth config is required")
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
	if err := c.AdminAuth.ValidateStartup(); err != nil {
		return err
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
	c.Redis.AddFlags(fss.FlagSet("redis"))
	c.AdminAuth.AddFlags(fss.FlagSet("admin-auth"))
	return fss
}

func New() *Config {
	//配置默认初始化
	return &Config{
		Log:       log.NewOptions(),
		Server:    options.NewServerOptions(),
		Registry:  options.NewRegistryOptions(),
		RPC:       options.NewRPCSecurityOptions(),
		Jwt:       options.NewJwtOptions(),
		Redis:     options.NewRedisOptions(),
		AdminAuth: NewAdminAuthOptions(),
	}
}
