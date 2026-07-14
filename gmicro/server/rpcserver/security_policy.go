package rpcserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

// SecurityPolicy defines the certificate and server identity requirements for
// internal gRPC mTLS.
type SecurityPolicy struct {
	CertFile   string `json:"cert-file,omitempty" mapstructure:"cert-file"`
	KeyFile    string `json:"key-file,omitempty" mapstructure:"key-file"`
	CAFile     string `json:"ca-file,omitempty" mapstructure:"ca-file"`
	ServerName string `json:"server-name,omitempty" mapstructure:"server-name"`
}

func NewSecurityPolicy() *SecurityPolicy {
	return &SecurityPolicy{}
}

func (p *SecurityPolicy) Validate() []error {
	if p == nil {
		return nil
	}

	var errs []error
	if (p.CertFile == "") != (p.KeyFile == "") {
		errs = append(errs, errors.New("rpc-security.cert-file and rpc-security.key-file must be configured together"))
	}
	return errs
}

// ValidateStartup validates a client-side policy for production startup.
func (p *SecurityPolicy) ValidateStartup() error {
	if err := p.validateFiles(); err != nil {
		return err
	}
	if p.ServerName == "" {
		return errors.New("rpc-security.server-name is required for production startup")
	}
	return nil
}

// ValidateServerStartup validates a server-side policy for production startup.
func (p *SecurityPolicy) ValidateServerStartup() error {
	return p.validateFiles()
}

func (p *SecurityPolicy) validateFiles() error {
	if p == nil {
		return errors.New("rpc-security config is required for production startup")
	}
	if p.CertFile == "" {
		return errors.New("rpc-security.cert-file is required for production startup")
	}
	if p.KeyFile == "" {
		return errors.New("rpc-security.key-file is required for production startup")
	}
	if p.CAFile == "" {
		return errors.New("rpc-security.ca-file is required for production startup")
	}
	return nil
}

func (p *SecurityPolicy) AddFlags(fs *pflag.FlagSet) {
	if fs == nil || p == nil {
		return
	}
	fs.StringVar(&p.CertFile, "rpc-security.cert-file", p.CertFile, "client/server certificate file for internal RPC mTLS")
	fs.StringVar(&p.KeyFile, "rpc-security.key-file", p.KeyFile, "client/server private key file for internal RPC mTLS")
	fs.StringVar(&p.CAFile, "rpc-security.ca-file", p.CAFile, "trusted CA certificate file for internal RPC mTLS")
	fs.StringVar(&p.ServerName, "rpc-security.server-name", p.ServerName, "expected TLS server name for internal RPC clients")
}

func (p *SecurityPolicy) LoadServerTLSConfig() (*tls.Config, error) {
	if err := p.ValidateServerStartup(); err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(p.CertFile, p.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load rpc server cert/key: %w", err)
	}

	clientCAs, err := loadCertPool(p.CAFile)
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

func (p *SecurityPolicy) LoadClientTLSConfig() (*tls.Config, error) {
	if err := p.ValidateStartup(); err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(p.CertFile, p.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load rpc client cert/key: %w", err)
	}

	rootCAs, err := loadCertPool(p.CAFile)
	if err != nil {
		return nil, fmt.Errorf("load rpc client root CA: %w", err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ServerName:   p.ServerName,
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
