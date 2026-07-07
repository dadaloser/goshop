package options

import (
	"errors"
	"fmt"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/pflag"
)

type JwtOptions struct {
	Realm      string        `json:"realm"       mapstructure:"realm"`
	Key        string        `json:"key"         mapstructure:"key"`
	Timeout    time.Duration `json:"timeout"     mapstructure:"timeout"`
	MaxRefresh time.Duration `json:"max-refresh" mapstructure:"max-refresh"`
}

func NewJwtOptions() *JwtOptions {
	return &JwtOptions{
		Realm:      "imooc",
		Key:        "imooc",
		Timeout:    24 * time.Hour,
		MaxRefresh: 24 * time.Hour,
	}
}

func (s *JwtOptions) Validate() []error {
	var errs []error

	if s.Key != "" && !govalidator.StringLength(s.Key, "6", "64") {
		errs = append(errs, fmt.Errorf("--secret-key must larger than 5 and little than 65"))
	}

	return errs
}

func (s *JwtOptions) ValidateStartup() error {
	if s.Key == "" {
		return errors.New("jwt.key is required")
	}
	if s.Key == "imooc" {
		return errors.New("jwt.key must not use the development default")
	}
	if !govalidator.StringLength(s.Key, "32", "64") {
		return errors.New("jwt.key must be between 32 and 64 characters for production startup")
	}
	if s.Timeout <= 0 {
		return errors.New("jwt.timeout must be positive")
	}
	if s.MaxRefresh <= 0 {
		return errors.New("jwt.max-refresh must be positive")
	}

	return nil
}

func (s *JwtOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		return
	}

	fs.StringVar(&s.Realm, "jwt.realm", s.Realm, "Realm name to display to the user.")
	fs.StringVar(&s.Key, "jwt.key", s.Key, "Private key used to sign jwt token.")
	fs.DurationVar(&s.Timeout, "jwt.timeout", s.Timeout, "JWT token timeout.")

	fs.DurationVar(&s.MaxRefresh, "jwt.max-refresh", s.MaxRefresh, ""+
		"This field allows clients to refresh their token until MaxRefresh has passed.")
}
