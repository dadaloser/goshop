package options

import (
	"testing"
	"time"
)

func TestRedisOptionsValidateStartupRejectsInsecureTLS(t *testing.T) {
	opts := NewRedisOptions()
	opts.SSLInsecureSkipVerify = true

	if err := opts.ValidateStartup(); err == nil {
		t.Fatal("ValidateStartup() error = nil, want insecure TLS error")
	}
}

func TestJwtOptionsValidateStartupRejectsDefaultKey(t *testing.T) {
	opts := NewJwtOptions()

	if err := opts.ValidateStartup(); err == nil {
		t.Fatal("ValidateStartup() error = nil, want default key error")
	}
}

func TestJwtOptionsValidateAllowsExternalSecretInjection(t *testing.T) {
	opts := NewJwtOptions()
	opts.Key = ""

	if errs := opts.Validate(); len(errs) != 0 {
		t.Fatalf("Validate() errors = %v, want none", errs)
	}
}

func TestMySQLOptionsValidateStartup(t *testing.T) {
	opts := NewMySQLOptions()
	opts.Username = "user"
	opts.Password = "password"
	opts.Database = "goshop"

	if err := opts.ValidateStartup(); err != nil {
		t.Fatalf("ValidateStartup() error = %v", err)
	}
}

func TestMySQLOptionsValidateAllowsExternalSecretInjection(t *testing.T) {
	opts := NewMySQLOptions()
	opts.Username = ""
	opts.Password = ""
	opts.Database = ""

	if errs := opts.Validate(); len(errs) != 0 {
		t.Fatalf("Validate() errors = %v, want none", errs)
	}
}

func TestNewMySQLOptionsDisablesAutoMigrateByDefault(t *testing.T) {
	opts := NewMySQLOptions()

	if opts.AutoMigrate {
		t.Fatal("NewMySQLOptions().AutoMigrate = true, want false")
	}
}

func TestEsOptionsValidateStartup(t *testing.T) {
	opts := NewEsOptions()
	opts.Scheme = "https"
	opts.UseSSL = true
	opts.Timeout = 3 * time.Second

	if err := opts.ValidateStartup(); err != nil {
		t.Fatalf("ValidateStartup() error = %v", err)
	}
}

func TestServerOptionsValidateStartupRejectsWildcardCORS(t *testing.T) {
	opts := NewServerOptions()
	opts.Middlewares = []string{"cors"}
	opts.CorsAllowOrigins = []string{"*"}

	if err := opts.ValidateStartup(); err == nil {
		t.Fatal("ValidateStartup() error = nil, want wildcard CORS error")
	}
}

func TestServerOptionsValidateStartupRejectsManagementPortEqualHTTPPort(t *testing.T) {
	opts := NewServerOptions()
	opts.HttpPort = 8049
	opts.ManagementPort = 8049

	if err := opts.ValidateStartup(); err == nil {
		t.Fatal("ValidateStartup() error = nil, want management-port conflict")
	}
}

func TestRPCSecurityOptionsValidateStartup(t *testing.T) {
	opts := NewRPCSecurityOptions()
	opts.CertFile = "client.crt"
	opts.KeyFile = "client.key"
	opts.CAFile = "ca.crt"
	opts.ServerName = "goshop.internal"

	if err := opts.ValidateStartup(); err != nil {
		t.Fatalf("ValidateStartup() error = %v", err)
	}
}

func TestRPCSecurityOptionsValidateServerStartupAllowsMissingServerName(t *testing.T) {
	opts := NewRPCSecurityOptions()
	opts.CertFile = "server.crt"
	opts.KeyFile = "server.key"
	opts.CAFile = "ca.crt"

	if err := opts.ValidateServerStartup(); err != nil {
		t.Fatalf("ValidateServerStartup() error = %v", err)
	}
}

func TestRPCSecurityOptionsValidateStartupRejectsMissingServerName(t *testing.T) {
	opts := NewRPCSecurityOptions()
	opts.CertFile = "client.crt"
	opts.KeyFile = "client.key"
	opts.CAFile = "ca.crt"

	if err := opts.ValidateStartup(); err == nil {
		t.Fatal("ValidateStartup() error = nil, want missing server name error")
	}
}
