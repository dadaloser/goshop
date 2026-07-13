package options

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

type RPCSecurityOptions struct {
	CertFile   string `json:"cert-file,omitempty" mapstructure:"cert-file"`
	KeyFile    string `json:"key-file,omitempty" mapstructure:"key-file"`
	CAFile     string `json:"ca-file,omitempty" mapstructure:"ca-file"`
	ServerName string `json:"server-name,omitempty" mapstructure:"server-name"`
}

func NewRPCSecurityOptions() *RPCSecurityOptions {
	return &RPCSecurityOptions{}
}

func (o *RPCSecurityOptions) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error
	if (o.CertFile == "") != (o.KeyFile == "") {
		errs = append(errs, errors.New("rpc-security.cert-file and rpc-security.key-file must be configured together"))
	}
	return errs
}

func (o *RPCSecurityOptions) ValidateStartup() error {
	if err := o.validateFiles(); err != nil {
		return err
	}
	if o.ServerName == "" {
		return errors.New("rpc-security.server-name is required for production startup")
	}
	return nil
}

func (o *RPCSecurityOptions) ValidateServerStartup() error {
	return o.validateFiles()
}

func (o *RPCSecurityOptions) validateFiles() error {
	if o == nil {
		return errors.New("rpc-security config is required for production startup")
	}
	if o.CertFile == "" {
		return errors.New("rpc-security.cert-file is required for production startup")
	}
	if o.KeyFile == "" {
		return errors.New("rpc-security.key-file is required for production startup")
	}
	if o.CAFile == "" {
		return errors.New("rpc-security.ca-file is required for production startup")
	}
	return nil
}

func (o *RPCSecurityOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil || o == nil {
		return
	}
	fs.StringVar(&o.CertFile, "rpc-security.cert-file", o.CertFile, "client/server certificate file for internal RPC mTLS")
	fs.StringVar(&o.KeyFile, "rpc-security.key-file", o.KeyFile, "client/server private key file for internal RPC mTLS")
	fs.StringVar(&o.CAFile, "rpc-security.ca-file", o.CAFile, "trusted CA certificate file for internal RPC mTLS")
	fs.StringVar(&o.ServerName, "rpc-security.server-name", o.ServerName, "expected TLS server name for internal RPC clients")
}

func (o *RPCSecurityOptions) LoadServerTLSConfig() (*tls.Config, error) {
	if err := o.ValidateServerStartup(); err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(o.CertFile, o.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load rpc server cert/key: %w", err)
	}

	clientCAs, err := loadCertPool(o.CAFile)
	if err != nil {
		return nil, fmt.Errorf("load rpc server client CA: %w", err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAs,
	}, nil
}

func (o *RPCSecurityOptions) LoadClientTLSConfig() (*tls.Config, error) {
	if err := o.ValidateStartup(); err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(o.CertFile, o.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load rpc client cert/key: %w", err)
	}

	rootCAs, err := loadCertPool(o.CAFile)
	if err != nil {
		return nil, fmt.Errorf("load rpc client root CA: %w", err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ServerName:   o.ServerName,
		Certificates: []tls.Certificate{cert},
		RootCAs:      rootCAs,
	}, nil
}

func loadCertPool(caFile string) (*x509.CertPool, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("append PEM certs from %s", caFile)
	}
	return pool, nil
}
