package options

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/pflag"
)

type EsOptions struct {
	Host                  string        `json:"host" mapstructure:"host"`
	Port                  string        `json:"port" mapstructure:"port"`
	Scheme                string        `json:"scheme" mapstructure:"scheme"`
	Username              string        `json:"username,omitempty" mapstructure:"username"`
	Password              string        `json:"password,omitempty" mapstructure:"password"`
	Timeout               time.Duration `json:"timeout" mapstructure:"timeout"`
	UseSSL                bool          `json:"use-ssl" mapstructure:"use-ssl"`
	SSLInsecureSkipVerify bool          `json:"ssl-insecure-skip-verify" mapstructure:"ssl-insecure-skip-verify"`
	DisableHealthcheck    bool          `json:"disable-healthcheck" mapstructure:"disable-healthcheck"`
}

func NewEsOptions() *EsOptions {
	return &EsOptions{
		Host:    "127.0.0.1",
		Port:    "9200",
		Scheme:  "http",
		Timeout: 5 * time.Second,
	}
}

func (e *EsOptions) Validate() []error {
	errs := []error{}
	if e.UseSSL && e.Scheme == "http" {
		errs = append(errs, errors.New("es.scheme must be https when es.use-ssl is true"))
	}
	return errs
}

func (e *EsOptions) ValidateStartup() error {
	if e.Host == "" {
		return errors.New("es.host is required")
	}
	port, err := strconv.Atoi(e.Port)
	if err != nil {
		return fmt.Errorf("es.port must be numeric: %w", err)
	}
	if port <= 0 || port > 65535 {
		return fmt.Errorf("es.port must be between 1 and 65535, got %d", port)
	}
	if e.Scheme != "http" && e.Scheme != "https" {
		return fmt.Errorf("es.scheme must be http or https, got %q", e.Scheme)
	}
	if e.SSLInsecureSkipVerify {
		return errors.New("es.ssl-insecure-skip-verify must be false for production startup")
	}
	if e.Timeout <= 0 {
		return errors.New("es.timeout must be positive")
	}

	return nil
}

func (e *EsOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&e.Host, "es.host", e.Host, ""+
		"es service host address. If left blank, the following related es options will be ignored.")

	fs.StringVar(&e.Port, "es.port", e.Port, ""+
		"es service port If left blank, the following related es options will be ignored..")

	fs.StringVar(&e.Scheme, "es.scheme", e.Scheme, "es connection scheme, http or https.")
	fs.StringVar(&e.Username, "es.username", e.Username, "optional username for es.")
	fs.StringVar(&e.Password, "es.password", e.Password, "optional password for es.")
	fs.DurationVar(&e.Timeout, "es.timeout", e.Timeout, "es client timeout.")
	fs.BoolVar(&e.UseSSL, "es.use-ssl", e.UseSSL, "use TLS for es connections.")
	fs.BoolVar(&e.SSLInsecureSkipVerify, "es.ssl-insecure-skip-verify", e.SSLInsecureSkipVerify,
		"allow insecure TLS certificate verification for es connections.")
	fs.BoolVar(&e.DisableHealthcheck, "es.disable-healthcheck", e.DisableHealthcheck,
		"disable es startup healthcheck.")
}
