package client

import (
	"fmt"
	"goshop/app/pkg/options"
	"goshop/gmicro/registry"
	"goshop/gmicro/registry/consul"

	consulapi "github.com/hashicorp/consul/api"
)

func NewConsulDiscovery(opts *options.RegistryOptions) (registry.Discovery, error) {
	if opts == nil {
		return nil, fmt.Errorf("registry options are required")
	}
	if opts.Address == "" || opts.Scheme == "" {
		return nil, fmt.Errorf("registry address and scheme are required")
	}

	conf := consulapi.DefaultConfig()
	conf.Address = opts.Address
	conf.Scheme = opts.Scheme

	cli, err := consulapi.NewClient(conf)
	if err != nil {
		return nil, err
	}
	return consul.New(cli, consul.WithHealthCheck(true)), nil
}
