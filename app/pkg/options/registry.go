package options

import (
	"fmt"
	"net"

	"goshop/pkg/errors"

	"github.com/spf13/pflag"
)

type RegistryOptions struct {
	Address string `mapstructure:"address" json:"address,omitempty"`
	Scheme  string `mapstructure:"scheme" json:"scheme,omitempty"`
}

func NewRegistryOptions() *RegistryOptions {
	return &RegistryOptions{
		Address: "192.168.1.92:8500",
		Scheme:  "http",
	}
}

func (o *RegistryOptions) Validate() []error {
	errs := []error{}
	if o.Address == "" {
		errs = append(errs, errors.New("registry.address is required"))
	} else if _, _, err := net.SplitHostPort(o.Address); err != nil {
		errs = append(errs, fmt.Errorf("registry.address must be host:port, got %q: %w", o.Address, err))
	}
	if o.Scheme == "" {
		errs = append(errs, errors.New("registry.scheme is required"))
	} else if o.Scheme != "http" && o.Scheme != "https" {
		errs = append(errs, fmt.Errorf("registry.scheme must be http or https, got %q", o.Scheme))
	}
	return errs
}

func (o *RegistryOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Address, "registry.address", o.Address, ""+
		"registry address, default is 127.0.0.1:8500")

	fs.StringVar(&o.Scheme, "registry.scheme", o.Scheme, ""+
		"registry scheme, default is http")
}
