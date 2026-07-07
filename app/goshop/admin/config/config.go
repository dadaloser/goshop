package config

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"goshop/app/pkg/options"
	"goshop/pkg/app"
	cliflag "goshop/pkg/common/cli/flag"
	"goshop/pkg/log"

	"github.com/spf13/pflag"
)

type Config struct {
	Log *log.Options `json:"log" mapstructure:"log"`

	Server    *options.ServerOptions   `json:"server" mapstructure:"server"`
	Registry  *options.RegistryOptions `json:"registry" mapstructure:"registry"`
	AdminAuth *AdminAuthOptions        `json:"admin-auth" mapstructure:"admin-auth"`
}

type AdminAuthOptions struct {
	Token       string   `json:"token" mapstructure:"token"`
	Role        string   `json:"role" mapstructure:"role"`
	Permissions []string `json:"permissions" mapstructure:"permissions"`
}

const (
	AdminRoleBasic        = "basic"
	AdminRoleAdmin        = "admin"
	AdminRolePrimaryAdmin = "primary_admin"
	AdminRoleSuperAdmin   = "super_admin"
)

var adminRoleLevels = map[string]int{
	AdminRoleBasic:        1,
	AdminRoleAdmin:        2,
	AdminRolePrimaryAdmin: 3,
	AdminRoleSuperAdmin:   4,
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

func (o *AdminAuthOptions) EffectivePermissions() []string {
	if o != nil && len(o.Permissions) > 0 {
		return normalizePermissions(o.Permissions)
	}
	return normalizePermissions(strings.Split(os.Getenv("GOSHOP_ADMIN_PERMISSIONS"), ","))
}

func (o *AdminAuthOptions) EffectiveRole() string {
	if o != nil && strings.TrimSpace(o.Role) != "" {
		return normalizeRole(o.Role)
	}
	return normalizeRole(os.Getenv("GOSHOP_ADMIN_ROLE"))
}

func (o *AdminAuthOptions) HasPermission(permission string) bool {
	permission = strings.TrimSpace(permission)
	if permission == "" {
		return true
	}
	for _, candidate := range o.EffectivePermissions() {
		if candidate == "*" || candidate == permission {
			return true
		}
		if strings.HasSuffix(candidate, ":*") {
			prefix := strings.TrimSuffix(candidate, "*")
			if strings.HasPrefix(permission, prefix) {
				return true
			}
		}
	}
	return false
}

func (o *AdminAuthOptions) HasRoleAtLeast(required string) bool {
	required = normalizeRole(required)
	if required == "" {
		return true
	}

	currentLevel, ok := adminRoleLevels[o.EffectiveRole()]
	if !ok {
		return false
	}
	requiredLevel, ok := adminRoleLevels[required]
	if !ok {
		return false
	}
	return currentLevel >= requiredLevel
}

func (o *AdminAuthOptions) HasAccess(permission, minRole string) bool {
	return o != nil && o.HasPermission(permission) && o.HasRoleAtLeast(minRole)
}

func normalizePermissions(values []string) []string {
	permissions := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		permission := strings.TrimSpace(value)
		if permission == "" {
			continue
		}
		if _, ok := seen[permission]; ok {
			continue
		}
		seen[permission] = struct{}{}
		permissions = append(permissions, permission)
	}
	return permissions
}

func normalizeRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func (o *AdminAuthOptions) Validate() []error {
	return nil
}

func (o *AdminAuthOptions) ValidateStartup() error {
	if o.EffectiveToken() == "" {
		return errors.New("admin-auth.token or GOSHOP_ADMIN_TOKEN is required")
	}
	if len(o.EffectivePermissions()) == 0 {
		return errors.New("admin-auth.permissions or GOSHOP_ADMIN_PERMISSIONS is required")
	}
	if _, ok := adminRoleLevels[o.EffectiveRole()]; !ok {
		return errors.New("admin-auth.role or GOSHOP_ADMIN_ROLE must be one of: basic, admin, primary_admin, super_admin")
	}
	return nil
}

func (o *AdminAuthOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		return
	}
	fs.StringVar(&o.Token, "admin-auth.token", o.Token, "shared token required for admin routes until full RBAC is enabled")
	fs.StringVar(&o.Role, "admin-auth.role", o.Role, "bootstrap admin role: basic, admin, primary_admin, or super_admin")
	fs.StringSliceVar(&o.Permissions, "admin-auth.permissions", o.Permissions, "permissions granted to the bootstrap admin token, for example user:list or user:*")
}

func (c *Config) Validate() []error {
	var errors []error
	errors = append(errors, c.Log.Validate()...)
	errors = append(errors, c.Server.Validate()...)
	errors = append(errors, c.Registry.Validate()...)
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
	c.AdminAuth.AddFlags(fss.FlagSet("admin-auth"))
	return fss
}

func New() *Config {
	//配置默认初始化
	return &Config{
		Log:       log.NewOptions(),
		Server:    options.NewServerOptions(),
		Registry:  options.NewRegistryOptions(),
		AdminAuth: NewAdminAuthOptions(),
	}
}
