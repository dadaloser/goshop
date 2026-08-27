package rpcserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
)

// SecurityPolicy defines the certificate and server identity requirements for
// internal gRPC mTLS.
type SecurityPolicy struct {
	CertFile                string
	KeyFile                 string
	CAFile                  string
	ServerName              string
	AllowedClientIdentities []string
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
	if err := p.validateFiles(); err != nil {
		return err
	}
	if len(p.AllowedClientIdentities) == 0 {
		return errors.New("rpc-security.allowed-client-identities is required for server startup")
	}
	seen := make(map[string]struct{}, len(p.AllowedClientIdentities))
	for _, identity := range p.AllowedClientIdentities {
		identity = strings.TrimSpace(identity)
		if identity == "" {
			return errors.New("rpc-security.allowed-client-identities must not contain empty identities")
		}
		if _, ok := seen[identity]; ok {
			return fmt.Errorf("rpc-security.allowed-client-identities contains duplicate %q", identity)
		}
		seen[identity] = struct{}{}
	}
	return nil
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
