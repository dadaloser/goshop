package config

import (
	"testing"

	"goshop/pkg/log"
)

func TestAdminAuthOptionsEffectiveConfirmationTokenFromEnv(t *testing.T) {
	t.Setenv("GOSHOP_ADMIN_CONFIRMATION_TOKEN", "confirm-secret")

	opts := &AdminAuthOptions{}
	if got := opts.EffectiveConfirmationToken(); got != "confirm-secret" {
		t.Fatalf("EffectiveConfirmationToken() = %q, want %q", got, "confirm-secret")
	}
}

func TestAdminAuthOptionsValidateStartup(t *testing.T) {
	t.Setenv("GOSHOP_ADMIN_TOKEN", "")

	tests := []struct {
		name    string
		opts    *AdminAuthOptions
		wantErr bool
	}{
		{
			name:    "missing token rejects",
			opts:    &AdminAuthOptions{},
			wantErr: true,
		},
		{
			name: "token-only break glass passes",
			opts: &AdminAuthOptions{Token: "secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.ValidateStartup()
			if tt.wantErr && err == nil {
				t.Fatal("ValidateStartup() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateStartup() error = %v", err)
			}
		})
	}
}

func TestConfigValidateStartupRequiresAdminAuth(t *testing.T) {
	cfg := &Config{Log: &log.Options{}}

	if err := cfg.ValidateStartup(); err == nil {
		t.Fatal("ValidateStartup() error = nil, want missing admin-auth error")
	}
}

func TestConfigValidateStartupDoesNotBypassAuthInDevelopment(t *testing.T) {
	cfg := &Config{Log: &log.Options{Development: true}}

	if err := cfg.ValidateStartup(); err == nil {
		t.Fatal("ValidateStartup() error = nil, want missing admin-auth error")
	}
}
