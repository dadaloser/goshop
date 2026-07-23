package options

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

type EmailOptions struct {
	Enabled      bool          `json:"enabled" mapstructure:"enabled"`
	Host         string        `json:"host" mapstructure:"host"`
	Port         int           `json:"port" mapstructure:"port"`
	Username     string        `json:"username" mapstructure:"username"`
	Password     string        `json:"-" mapstructure:"password"`
	From         string        `json:"from" mapstructure:"from"`
	CodeTTL      time.Duration `json:"code-ttl" mapstructure:"code-ttl"`
	SendInterval time.Duration `json:"send-interval" mapstructure:"send-interval"`
}

func NewEmailOptions() *EmailOptions {
	return &EmailOptions{Port: 587, CodeTTL: 10 * time.Minute, SendInterval: time.Minute}
}
func (o *EmailOptions) Validate() []error {
	if o == nil || !o.Enabled {
		return []error{}
	}
	errs := []error{}
	if o.Host == "" || o.From == "" {
		errs = append(errs, errors.New("email host and from are required"))
	}
	if o.CodeTTL <= 0 || o.SendInterval <= 0 {
		errs = append(errs, errors.New("email code durations must be positive"))
	}
	return errs
}
func (o *EmailOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.Enabled, "email.enabled", o.Enabled, "enable SMTP email verification")
	fs.StringVar(&o.Host, "email.host", o.Host, "SMTP host")
	fs.IntVar(&o.Port, "email.port", o.Port, "SMTP port")
	fs.StringVar(&o.Username, "email.username", o.Username, "SMTP username")
	fs.StringVar(&o.Password, "email.password", o.Password, "SMTP password")
	fs.StringVar(&o.From, "email.from", o.From, "verification sender")
}
