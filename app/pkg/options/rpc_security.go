package options

import (
	"goshop/gmicro/server/rpcserver"

	"github.com/spf13/pflag"
)

// RPCSecurityOptions is the application configuration DTO for internal RPC mTLS.
type RPCSecurityOptions struct {
	CertFile                string   `json:"cert-file,omitempty" mapstructure:"cert-file"`
	KeyFile                 string   `json:"key-file,omitempty" mapstructure:"key-file"`
	CAFile                  string   `json:"ca-file,omitempty" mapstructure:"ca-file"`
	ServerName              string   `json:"server-name,omitempty" mapstructure:"server-name"`
	AllowedClientIdentities []string `json:"allowed-client-identities,omitempty" mapstructure:"allowed-client-identities"`
}

func NewRPCSecurityOptions() *RPCSecurityOptions {
	return &RPCSecurityOptions{}
}

// ToPolicy creates a runtime policy at the application/framework boundary.
func (o *RPCSecurityOptions) ToPolicy() *rpcserver.SecurityPolicy {
	if o == nil {
		return nil
	}
	return &rpcserver.SecurityPolicy{
		CertFile:                o.CertFile,
		KeyFile:                 o.KeyFile,
		CAFile:                  o.CAFile,
		ServerName:              o.ServerName,
		AllowedClientIdentities: append([]string(nil), o.AllowedClientIdentities...),
	}
}

func (o *RPCSecurityOptions) Validate() []error { return o.ToPolicy().Validate() }

func (o *RPCSecurityOptions) ValidateStartup() error { return o.ToPolicy().ValidateStartup() }

func (o *RPCSecurityOptions) ValidateServerStartup() error {
	return o.ToPolicy().ValidateServerStartup()
}

func (o *RPCSecurityOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil || o == nil {
		return
	}
	fs.StringVar(&o.CertFile, "rpc-security.cert-file", o.CertFile, "client/server certificate file for internal RPC mTLS")
	fs.StringVar(&o.KeyFile, "rpc-security.key-file", o.KeyFile, "client/server private key file for internal RPC mTLS")
	fs.StringVar(&o.CAFile, "rpc-security.ca-file", o.CAFile, "trusted CA certificate file for internal RPC mTLS")
	fs.StringVar(&o.ServerName, "rpc-security.server-name", o.ServerName, "expected TLS server name for internal RPC clients")
	fs.StringSliceVar(&o.AllowedClientIdentities, "rpc-security.allowed-client-identities", o.AllowedClientIdentities,
		"exact URI SAN or DNS SAN identities authorized to call this gRPC server")
}
